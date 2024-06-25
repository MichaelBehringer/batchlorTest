package vnc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/errors"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/kuber"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/models"
	"gitea.hama.de/LFS/lfsx-web/controller/pkg/utils"
	"github.com/lesismal/nbio"
	"github.com/lesismal/nbio/logging"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

// VncProxy handles the proxying of the TCP VNC Socket to
// a WebSocket connection that NoVNC can use.
//
// In addition to that it also provides a way to communicate with the
// LFS.X API and the host API of that pod.
//
// Based on the currently authenticated user the IP address of the
// already assigned pod will be fetched and a WebSocket connection
// is served for the user.
// If no User Pod is available a new Pod will be assigned for the user.
type VncProxy struct {

	// Kubernetes client
	kuber *kuber.Kuber

	// Context used for this client
	baseContext context.Context
	// Cancel function to cancel the context
	cancelBaseContext context.CancelFunc

	// NBIO Engine used to connect to the VNC backend server
	engine *nbio.Engine

	// Manager for the Ping Pong messages
	pingPongMgr ClientMgr

	// Configuration of the app
	config *models.AppConfig

	// A list of peers that are currently opened index by the login username
	// and the selected db
	peer map[string]*peer
	// Sync to access the peers
	peerSync sync.RWMutex
}

// NewVncService initializes a new service to proxy
// a "TCP VNC connection" <=> "WebSocket connection".
//
// When the given context is closed, all ressources are freeded
// up created by this method.
func NewVncProxy(ctx context.Context, kuber *kuber.Kuber, config *models.AppConfig) (*VncProxy, error) {

	// Set default logger that nbio should use
	logging.DefaultLogger = newNbioLogger()

	// Create context
	baseContext, cancelBaseContext := context.WithCancel(ctx)

	// Start an engine that is used for all VNC connections
	engine := nbio.NewEngine(nbio.Config{})
	go func() {
		<-baseContext.Done()
		engine.Stop()
	}()

	vnc := &VncProxy{
		engine:            engine,
		pingPongMgr:       *NewClientMgr(KeepAliveTimeout, baseContext),
		peer:              make(map[string]*peer),
		kuber:             kuber,
		config:            config,
		baseContext:       baseContext,
		cancelBaseContext: cancelBaseContext,
	}

	// Start the engine
	err := engine.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start engine for VNC connections: %s", err)
	}
	go vnc.pingPongMgr.Run()

	// Assign the engine methods
	engine.OnData(vnc.onEngineMessage)
	engine.OnClose(vnc.onEngineClose)

	return vnc, nil
}

// Proxy proxies the given request to a VNC instance for this
// specific user.
func (vnc *VncProxy) Proxy(w http.ResponseWriter, r *http.Request, user *models.User, useGuacamole bool, vncSettings VncConnectionSettings) error {

	// Validate the user request
	if err := vnc.validateUserRequest(user); err != nil {
		return err
	}

	peer := NewPeer(user, vnc.onPeerDisconnect)

	// Create WebSocket upgrader
	upgrader := vnc.newUpgrader(peer)
	upgrader.OnClose(func(c *websocket.Conn, err error) {
		logger.Debug("Closed client VNC connection in ws upgrade sequence")
		peer.Close(err, 1)
	})

	// Upgrade to a WebSocket connection
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Warning("Cannot upgrade to ws: %s", err)
		return err
	}

	// Setup websocket connection
	wsConn.SetReadDeadline(time.Now().Add(KeepAliveTimeout))
	vnc.pingPongMgr.Add(wsConn)
	wsConn.OnClose(func(c *websocket.Conn, err error) {
		logger.Debug("Client closed connection for VNC")
		peer.Close(err, 1)
	})

	// Create TCP connection to the VNC backend
	podAddr, wasPodNewlyCreated, err := vnc.getVncAddress(user, useGuacamole)
	if err != nil {
		logger.Warning("Cannot get IP of pod: %s", err)
		return err
	}
	var vncCon *nbio.Conn = nil
	var guacamoleCon *net.Conn = nil

	// Get the remote pod ip (udp = addres + port - that's not the protocol :)
	ip, err := net.ResolveUDPAddr("udp", podAddr.String())
	if err != nil {
		logger.Warning("Could not determine the pods IP address from the connection: %s", err)
		return err
	}
	logger.Debug("Using pod address: %s", ip.IP)

	// Connect to the VNC backend
	if !useGuacamole {
		vncCon, err = vnc.connectToVnc(podAddr)
		if err != nil {
			logger.Warning("Cannot connect to VNC backend %s", err)
			return err
		}
		vncCon.SetSession(peer)
	}

	// Get the reverse proxy for the API endpoints
	lfsxAPI, hostAPI, err := vnc.getHTTPProxy(ip)
	if err != nil {
		logger.Warning("Cannot create reverse proxy for endpoint: %s", err)
	}

	// Apply settings
	if err := vnc.applyVncSettings(vncSettings, podAddr, wasPodNewlyCreated); err != nil {
		logger.Warning("Failed to apply scaling factor: %s", err)
	}

	// Get the WebSocket connection to the LFS.X
	hostPeer, err := NewLfsxPeer(peer, vnc.baseContext, ip, &vnc.pingPongMgr, nil)
	if err != nil {
		logger.Warning("Cannot connect to the LFS.X WebSocket. LFS.X specific functions won't be avaialable: %s", err)
	} else {
		peer.lfsxPeer = hostPeer
		updateChan := hostPeer.RegisterObserver()

		// Send the login request
		go func() {
			hostPeer.SendMessageToLFS(models.NewWebSocketData(0,
				models.NewLoginRequest(user.Username, user.DbPassword, user.DatabaseStr),
			))
		}()

		// Wait until the LFS.X does boot up
		select {
		case <-time.After(20 * time.Second):
			logger.Debug("LFS.X did not boot up within 20 seconds. Continuing anyway")
		case up := <-updateChan:
			if up.Message.Type != "LfsStartup" {
				logger.Debug("Received a WebSocket message from the LFS.X from type %q but expected %q", up.Message.Type, "LfsStartup")
				// It doesn't matter because the LFS.X doesn't send any messages before it's bootstrapped
			}
		}
	}

	// Create TCP connection to gucd
	if useGuacamole {
		guacamoleCon, err = vnc.connectToGuacamole(podAddr)
		if err != nil {
			logger.Warning("Cannot connect to guacd %s", err)
			return err
		}
	}

	// Set connections
	peer.SetConnections(wsConn, vncCon, guacamoleCon, lfsxAPI, hostAPI)

	// Add to list
	vnc.peerSync.Lock()
	defer vnc.peerSync.Unlock()
	if _, doesExist := vnc.peer[user.Identifier()]; !doesExist {
		// Set the peer
		vnc.peer[user.Identifier()] = peer
	} else {
		// Peer does already exists -> return error message
		peer.Close(fmt.Errorf("USER_ALREADY_EXISTS"), 0)
		return errors.NewError("USER_ALREADY_EXISTS", 409)
	}

	if useGuacamole {
		// Create a new Guacamole stram
		if err := peer.proxyGuacamole(r.URL.Query().Get("quality")); err != nil {
			logger.Error("Failed to create proxy to guacd: %s", err)
		}
	} else {
		// Because we've lost the initial "Hello" Message from the Server (the connection was not setup already) we need to send it again.
		// Take a look at the RFB protocol for more informations
		go func() {
			peer.OnTargetMessage(vncCon, []byte("RFB 003.008\n"))
		}()
	}

	// Print info
	logger.Info("Opened connection for user %q (%s): %s", user.Username, user.Database, wsConn.RemoteAddr().String())
	return nil
}

// ProxyHostWebsocket establish a proxy WebSocket connection to the LFS.X. This connection can also
// be used from the controller
func (vnc *VncProxy) ProxyLfsxWebsocket(user *models.User, w http.ResponseWriter, r *http.Request) error {
	vnc.peerSync.RLock()
	u, doesExist := vnc.peer[user.Identifier()]
	vnc.peerSync.RUnlock()

	if !doesExist || !u.IsReady() || u.lfsxPeer == nil {
		return errors.NewError("No VNC connection established for your user", 424)
	}

	return u.lfsxPeer.ProxyHostWebsocket(w, r)
}

// IsUserConnected returns weather the given user is connected to the API
func (vnc *VncProxy) IsUserConnected(user *models.User) bool {
	vnc.peerSync.RLock()
	_, doesExist := vnc.peer[user.Identifier()]
	vnc.peerSync.RUnlock()

	return doesExist
}

// validateUserRequest validates if the request from the given user is
// valid.
// This does include for example that only one connection to the proxy does
// exist
func (vnc *VncProxy) validateUserRequest(user *models.User) error {

	// Check if a connection for the user does already exist
	if vnc.IsUserConnected(user) {
		return errors.NewError("USER_ALREADY_EXISTS", 409)
	}

	return nil
}

// Probe validates the user requests and creates a new pod
// if no one does already exist.
// This method will block until the pod was created
func (vnc *VncProxy) Probe(user *models.User, settings VncConnectionSettings) error {
	// Validate the user request
	if err := vnc.validateUserRequest(user); err != nil {
		return err
	}

	// In debug mode don't even try to get pod
	if vnc.config.DevConfig.VncAddress != "" {
		return nil
	}

	// Create pod
	pod, wasCreated, err := vnc.kuber.GetPodByUser(user)
	if err != nil {
		logger.Warning("Failed to create pod: %s", err)
		return errors.NewError("Failed to create pod", 500)
	}

	// Apply settings
	n, err := net.ResolveTCPAddr("tcp", pod.Status.PodIP+":1")
	if err != nil {
		return err
	}
	if err := vnc.applyVncSettings(settings, n, wasCreated); err != nil {
		logger.Warning("Failed to apply scaling factor: %s", err)
	}

	return nil
}

// onPeerDisconnect is called when a peer goes offline
func (vnc *VncProxy) onPeerDisconnect(peer *peer, err error, fromWho int) {
	vnc.pingPongMgr.Delete(peer.source)
	logger.Info("Closed connection for user %q", peer.user.Username)

	// Remove peer from map
	vnc.peerSync.Lock()
	delete(vnc.peer, peer.user.Identifier())
	vnc.peerSync.Unlock()
}

// getVncAddress gets the address of the pod to connect to.
// If no pod does already exists, a new Pod will be created.
//
// This method does block until the pod was created.
//
// The second parameter states weather a new pod or an already existing one was assigned
func (vnc *VncProxy) getVncAddress(user *models.User, useGuacamole bool) (netAddr net.Addr, newPodCreated bool, err error) {
	// Get a static IP address for vnc connection in development mode
	addr := ""
	if useGuacamole {
		addr = vnc.config.DevConfig.GuacamoleAddress
	} else {
		addr = vnc.config.DevConfig.VncAddress
	}

	if addr == "" {
		// Get the address of the pod
		pod, newCreated, err := vnc.kuber.GetPodByUser(user)
		if err != nil {
			return nil, newCreated, fmt.Errorf("failed to create pod: %s", err)
		}
		newPodCreated = newCreated

		// Add port number to connection
		if useGuacamole {
			addr = pod.Status.PodIP + ":4822"
		} else {
			addr = pod.Status.PodIP + ":5910"
		}

	} else {
		logger.Info("Using predefined address instead of pod address: %q", addr)
		newPodCreated = true
	}

	netAddr, err = net.ResolveTCPAddr("tcp", addr)
	return
}

// connectToVnc creates a connection to the VNC backend pod
// assigned for the user.
func (vnc *VncProxy) connectToVnc(addr net.Addr) (*nbio.Conn, error) {

	c, err := nbio.DialTimeout(addr.Network(), addr.String(), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("%q: %s", addr, err)
	}

	// Add the connection to the engine
	c, err = vnc.engine.AddConn(c)
	if err != nil {
		return nil, fmt.Errorf("failed to add VNC connection to the shared engine")
	}

	c.SetKeepAlivePeriod(KeepAliveTimeout / 2)
	c.SetKeepAlive(true)

	return c, nil
}

// connectToGuacamole establishes a connection to guacd.
//
// Instead of a nbio.conn this function returns a "normal" net.conn.
// That's because we need to parse and read the messages send by guacd
// sequentiell byte by byte.
// With nbio thats not possible because the library is designed for efficient
// async and buffer reuse.
// Even when syncing the incoming messages and using a single channel reader
// the messages doesn't arriver correctly and cannot be parsed therefore correclty
// for the client
func (vnc *VncProxy) connectToGuacamole(addr net.Addr) (*net.Conn, error) {
	c, err := net.Dial(addr.Network(), addr.String())
	if err != nil {
		return nil, fmt.Errorf("%q: %s", addr, err)
	}

	return &c, nil
}

// newUpgrader creates an upgrader for the WebSocket connection
func (vnc *VncProxy) newUpgrader(peer *peer) *websocket.Upgrader {
	u := websocket.NewUpgrader()

	u.KeepaliveTime = KeepAliveTimeout

	// Handle pong messages
	u.OnMessage(func(c *websocket.Conn, mt websocket.MessageType, b []byte) {
		c.SetDeadline(time.Now().Add(KeepAliveTimeout))
		peer.OnSourceMessage(c, mt, b)
	})
	u.SetPongHandler(func(c *websocket.Conn, s string) {
		c.SetDeadline(time.Now().Add(KeepAliveTimeout))
	})

	return u
}

// onEngineMessage handles an incoming message of an connection created
// by the global engine
func (vnc *VncProxy) onEngineMessage(c *nbio.Conn, data []byte) {
	// Sleep some time when no connection is available on connection because
	// the connection is set AFTER the tcp connection was established
	for i := 0; i < 3; i++ {
		if peer, ok := c.Session().(*peer); ok {
			peer.OnTargetMessage(c, data)
			return
		}

		logger.Trc("Sleeping 5 milliseconds because session was not set already")
		time.Sleep(5 * time.Millisecond)
	}

	logger.Debug("No peer session available for engine on message")
}

// onEngineClose handles the closing of a connection created by the
// global engine
func (vnc *VncProxy) onEngineClose(c *nbio.Conn, err error) {
	if peer, ok := c.Session().(*peer); ok {
		peer.Close(err, 2)
	} else {
		logger.Debug("No peer session available for engine on close")
	}
}

// getHTTPProxy builds up a HTTP reverse proxy to the LFSX API and the
// host API
func (vnc *VncProxy) getHTTPProxy(ip *net.UDPAddr) (lfsx *httputil.ReverseProxy, host *httputil.ReverseProxy, err error) {
	remoteLfsURL, err := url.Parse(fmt.Sprintf("http://%s:8888", ip.IP))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse the url for the LFS.X endpoint 'http://%s:8888': %s", ip.IP, err)
	}
	remoteHostURL, err := url.Parse(fmt.Sprintf("http://%s:%d", ip.IP, utils.GetEnvInt("APP_LFS_API_PORT", 4021)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse the url for the host endpoint 'http://%s:%d': %s", ip.IP, utils.GetEnvInt("APP_LFS_API_PORT", 4021), err)
	}

	return httputil.NewSingleHostReverseProxy(remoteLfsURL), httputil.NewSingleHostReverseProxy(remoteHostURL), nil
}

// ProxyLfsxRequest proxies the given request to the LFSX API endpoint
func (vnc *VncProxy) ProxyLfsxRequest(user *models.User, response http.ResponseWriter, request *http.Request) error {
	vnc.peerSync.RLock()
	defer vnc.peerSync.RUnlock()

	if user, doesExist := vnc.peer[user.Identifier()]; doesExist {
		// Check for matching db
		requestedDb := request.Header.Get("Db")
		if requestedDb != "" && !strings.EqualFold(requestedDb, user.user.DatabaseStr) {
			return errors.NewError("Database missmatch between VNC db and requested db", 409)
		}

		user.lfsxAPI.ServeHTTP(response, request)
		return nil
	}

	return errors.NewError("User is not connected to a LFS.X instance", 421)
}

// ProxyHostRequest proxies the given request to the LFSX API endpoint
func (vnc *VncProxy) ProxyHostRequest(user *models.User, response http.ResponseWriter, request *http.Request) error {
	vnc.peerSync.RLock()
	defer vnc.peerSync.RUnlock()

	if user, doesExist := vnc.peer[user.Identifier()]; doesExist {
		user.hostAPI.ServeHTTP(response, request)
		return nil
	}

	return errors.NewError("User is not connected to an LFS.X instance", 421)
}

// applyVncSettings applies the given VNC specific settings for the pod.
// It may be possible that the LFS.X may be restarted within this function
func (vnc *VncProxy) applyVncSettings(settings VncConnectionSettings, podAdr net.Addr, wasNewlyCreated bool) error {

	// Get the remote pod ip (udp = addres + port - that's not the protocol :)
	ip, err := net.ResolveUDPAddr("udp", podAdr.String())
	if err != nil {
		logger.Warning("Could not determine the pods IP address from the connection: %s", err)
		return err
	}
	baseURL := fmt.Sprintf("http://%s:%d/api", ip.IP.String(), utils.GetEnvInt("APP_LFS_API_PORT", 4021))

	// Apply the scaling factor provided by the user by calling the (hard) scaling endpoint
	// of the LFS container.
	// This DOES restart the LFS.X
	if wasNewlyCreated && settings.Scaling != 100 && settings.Scaling != 0 {
		// Send a request to the host
		client := http.Client{Timeout: 5 * time.Second}
		body, err := json.Marshal(struct {
			Factor int `json:"factor"`
		}{Factor: settings.Scaling})
		if err != nil {
			return fmt.Errorf("failed to unmarshal %s", err)
		}

		req, _ := http.NewRequest("POST", baseURL+"/vnc/scale/hard", bytes.NewReader(body))
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request to LFS.X host: %s", err)
		}

		// Expecting a 200 result code
		bodyR, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Warning("read body: %s", err)
		}

		// Check status
		if resp.StatusCode == 200 || resp.StatusCode == 207 {
			logger.Debug("Result from applying host scaling factor: %q", bodyR)
		} else {
			return fmt.Errorf("%d: %s", resp.StatusCode, bodyR)
		}
	}

	return nil
}
