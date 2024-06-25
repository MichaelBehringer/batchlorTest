package vnc

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/models"
	"github.com/lesismal/nbio/nbhttp"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

// lfsxPeer represents a WebSocket proxy peer with a
// websocket connection to the client and a
// LFS.X backend connection
type lfsxPeer struct {

	// WebSocket connection of the client
	source *websocket.Conn
	// Mutext to synchronize the connection access
	sourceSync sync.Mutex

	// Websocket backend connection to the LFS.X
	target *websocket.Conn
	// The raw IP address of the remote pod to connect to
	targetIp *net.UDPAddr
	// Mutext to synchronize the connection access
	targetSync sync.Mutex

	// The root VNC peer
	root *peer

	// The base context to use
	baseContext   context.Context
	cancelContext context.CancelFunc

	// Ready states if BOTH connections are setup for
	// further use
	ready atomic.Bool

	// If this peer should not reconnect again
	wasIntentiallyClosed atomic.Bool

	// Function that is called when this peer goes offline
	onDisconnect func(*lfsxPeer, error, int)

	// Engine to use
	engine *nbhttp.Engine

	// The ping pong manager
	pingPong *ClientMgr

	// All observers of the update chanel
	observers    []chan Update
	observerLock sync.RWMutex
}

// Update is used to notify all observers for a new Message from the WebSocket.
// This can either be the client or the LFS.X. See the field 'FromLfsx' if it's
// coming from the LFS.X
type Update struct {

	// The message that was send from the client
	Message models.WebSocketMessage

	// The unique ID of the upper WebSocket message (data)
	ID int

	// The ID of the message to which a reply was send
	ResponseTo int

	// Toggles weather the message came from the LFS.X
	FromLfsx bool
}

// NewLfsxPeer creates an empty peer for the given root per with only
// the WebSocket connection to the LFS.X.
//
// For correclty using this peer you have to call also the function
// "SetClientConnection()" after the client did connect.
//
// This method does block until the LFS.X connection could be established.
func NewLfsxPeer(root *peer, ctx context.Context, targetIp *net.UDPAddr, pingPong *ClientMgr, onDisconnect func(*lfsxPeer, error, int)) (*lfsxPeer, error) {

	// Branch a new context from the base context that is used for canceling all connections
	baseCtx, cancelBaseCtx := context.WithCancel(ctx)

	// Create engine to use
	engine := nbhttp.NewEngine(nbhttp.Config{Context: baseCtx})
	if err := engine.Start(); err != nil {
		cancelBaseCtx()
		return nil, fmt.Errorf("failed to start nbio engine: %s", err)
	}
	go func() {
		<-baseCtx.Done()
		logger.Debug("Stopping engine because of canceled context")
		engine.Stop()
	}()

	rtc := &lfsxPeer{
		root:          root,
		onDisconnect:  onDisconnect,
		baseContext:   baseCtx,
		cancelContext: cancelBaseCtx,
		engine:        engine,
		targetIp:      targetIp,
		pingPong:      pingPong,
	}

	// Connect to the LFS.X backend
	if err := rtc.connectToLfsxSocket(); err != nil {
		return nil, err
	}

	return rtc, nil
}

// ProxyHostWebsocket establish a proxy WebSocket connection to the LFS.X. This connection can also
// be used from the controller
func (p *lfsxPeer) ProxyHostWebsocket(w http.ResponseWriter, r *http.Request) error {

	// Create WebSocket upgrader
	upgrader := p.newUpgraderForClient()

	// Upgrade to a WebSocket connection
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Warning("Cannot upgrade to ws: %s", err)
		return err
	}

	// Setup websocket connection
	wsConn.SetReadDeadline(time.Now().Add(KeepAliveTimeout))
	p.pingPong.Add(wsConn)
	p.source = wsConn
	p.ready.Store(true)

	return nil
}

// connectToLfsxSocket establishes a connection to the LFS.X WebSocket running
// on the host.
// This method does block until a WebSocket connection could be established
func (p *lfsxPeer) connectToLfsxSocket() error {
	p.targetSync.Lock()
	defer p.targetSync.Unlock()

	if p.baseContext.Err() != nil {
		// Base Context was canceled
		return fmt.Errorf("not starting WebSocket to LFS.X: base context already canceled")
	}

	// Setup dialer
	dialer := websocket.Dialer{
		Engine:      p.engine,
		Upgrader:    p.newUpgraderForLfsx(),
		DialTimeout: time.Second * 5,
	}

	// Try to open the WebSocket connection within 26 seconds
	for i := 0; i < 13; i++ {
		// Check if the user still wants to connect (in addition to the channeld listener below)
		if p.baseContext.Err() != nil {
			logger.Debug("Not trying to connect to LFS.X again because context was canceled")
			return fmt.Errorf("context canceled")
		}

		// Open connection
		con, _, err := dialer.Dial(fmt.Sprintf("ws://%s:8888/kubernetes", p.targetIp.IP), nil)
		if err == nil {
			p.target = con
			break
		}

		// The LFS.X was probably not booted up -> try again in 2 seconds
		logger.Trc("Failed to connect to LFS.X. Trying again in 2 seconds: %s", err)
		select {
		case <-time.After(2 * time.Second):
			continue
		case <-p.baseContext.Done():
			logger.Debug("Not trying to connect to LFS.X again because context was canceled")
			return fmt.Errorf("context canceled")
		}
	}

	// No connection to the LFS.X could be established
	if p.target == nil {
		return fmt.Errorf("unable to connect to WebSocket")
	}

	p.target.SetReadDeadline(time.Now().Add(KeepAliveTimeout))
	p.pingPong.Add(p.target)

	logger.Debug("Connected to the LFS.X")
	return nil
}

// notifyForUpdates notifies all observer for an update
func (p *lfsxPeer) notifyForUpdates(fromLfsx bool, data *models.WebSocketData) {
	p.observerLock.RLock()
	defer p.observerLock.RUnlock()

	for _, obs := range p.observers {

		// Send an own message for every message in the upper data struct
		for i := range data.Messages {
			msg := Update{Message: data.Messages[i], FromLfsx: fromLfsx, ID: data.ID, ResponseTo: data.ResponseTo}
			go func(c chan Update) {
				c <- msg
			}(obs)
		}
	}
}

// RegisterObserver returns a new channel that is filled when an update
// of the data occur.
// You can check the models.Update methods to get more exact update details
func (p *lfsxPeer) RegisterObserver() chan Update {
	p.observerLock.Lock()
	defer p.observerLock.Unlock()

	c := make(chan Update)
	p.observers = append(p.observers, c)
	return c
}

// RemoveObserver removes the given observer from the internal observers
// lists and closed the channel
func (p *lfsxPeer) RemoveObserver(c chan Update) {
	p.observerLock.Lock()
	defer p.observerLock.Unlock()

	// Find the observer and remove it
	for i := range p.observers {
		if p.observers[i] == c {
			p.observers = append(p.observers[:i], p.observers[i+1:]...)
			close(c)
			break
		}
	}
}

// newUpgraderForLfsx creates an upgrader for the WebSocket connection to the LFS.X
func (p *lfsxPeer) newUpgraderForLfsx() *websocket.Upgrader {
	u := websocket.NewUpgrader()

	// Handle incoming messaged
	u.OnMessage(func(c *websocket.Conn, mt websocket.MessageType, b []byte) {
		c.SetDeadline(time.Now().Add(KeepAliveTimeout))
		p.OnTargetMessage(c, mt, b)
	})

	u.SetPongHandler(func(c *websocket.Conn, s string) {
		c.SetDeadline(time.Now().Add(KeepAliveTimeout))
	})

	u.SetCloseHandler(func(c *websocket.Conn, i int, s string) {
		p.pingPong.Delete(c)
		p.tryToReconnectToLfsx()
	})

	u.OnClose(func(c *websocket.Conn, err error) {
		logger.Debug("Closed WebSocket connection to the LFS.X (server)")
		p.pingPong.Delete(c)
		p.tryToReconnectToLfsx()
	})

	return u
}

// newUpgraderForClient creates an upgrader for the WebSocket client connecting to the controller
func (p *lfsxPeer) newUpgraderForClient() *websocket.Upgrader {
	u := websocket.NewUpgrader()

	// Handle incoming messages
	u.OnMessage(func(c *websocket.Conn, mt websocket.MessageType, b []byte) {
		c.SetDeadline(time.Now().Add(KeepAliveTimeout))
		p.OnSourceMessage(c, mt, b)
	})

	u.SetPongHandler(func(c *websocket.Conn, s string) {
		c.SetDeadline(time.Now().Add(KeepAliveTimeout))
	})

	u.SetCloseHandler(func(c *websocket.Conn, i int, s string) {
		p.pingPong.Delete(c)
	})

	u.OnClose(func(c *websocket.Conn, err error) {
		logger.Debug("Closed WebSocket connection to the LFS.X (client)")
		p.pingPong.Delete(c)

		p.sourceSync.Lock()
		p.source = nil
		p.sourceSync.Unlock()
	})

	return u
}

// tryToReconnectToLfsx tries to reconnect to the LFS.X
func (p *lfsxPeer) tryToReconnectToLfsx() {
	if p.wasIntentiallyClosed.Load() {
		logger.Debug("Connection to LFS.X WebSocket was intentially closed. Not trying to reconnect")
	} else {
		logger.Debug("Trying to reconnect to the LFS.X in 5 seconds")

		// Set connection to nil
		p.targetSync.Lock()
		p.target = nil
		p.targetSync.Unlock()

		go func() {
			select {
			case <-time.After(5 * time.Second):
				if err := p.connectToLfsxSocket(); err != nil {
					logger.Debug("Failed to reconnect to the LFS.X WebSocket: %s", err)
					p.tryToReconnectToLfsx()
				}
			case <-p.baseContext.Done():
				logger.Debug("Not rescheduling an reconnect because context was canceled")
			}
		}()
	}
}

// SetClientConnection is setting the client connection
// and making it so ready for the "full" usage
func (p *lfsxPeer) SetClientConnection(source *websocket.Conn) {
	p.source = source

	// Make both peers ready
	p.ready.Store(true)
}

// Close close both websocket connections.
//
// You can provide an error message (close reason) and a toggle from which
// side the close event came (0 = unknown, 1 = source, 2 = target)
func (p *lfsxPeer) Close(err error, fromWhich int) {
	p.sourceSync.Lock()
	defer p.sourceSync.Unlock()
	p.targetSync.Lock()
	defer p.targetSync.Unlock()

	if p.wasIntentiallyClosed.Load() || p.baseContext.Err() != nil {
		logger.Debug("Peer was already closed. Not closing...")
		return
	} else {
		p.wasIntentiallyClosed.Store(true)
	}

	if p.source != nil && fromWhich != 1 {
		// Close the WebSocket with a reason if one was given
		if err != nil {
			var codeBytes = make([]byte, 2)
			binary.BigEndian.PutUint16(codeBytes, 1008)
			codeBytes = append(codeBytes, err.Error()...)
			p.source.WriteMessage(websocket.CloseMessage, codeBytes)
		}

		p.source.Close()
	}

	if p.target != nil && fromWhich != 2 {
		p.target.Close()
	}

	// Stop engine
	p.cancelContext()

	//p.onDisconnect(p, err, fromWhich)
}

// SendMessageToLFS sends the given message to the LFS.X WebSocket endpoint
func (p *lfsxPeer) SendMessageToLFS(msg models.WebSocketData) {
	p.targetSync.Lock()
	defer p.targetSync.Unlock()

	if p.wasIntentiallyClosed.Load() || p.baseContext.Err() != nil || p.target == nil {
		logger.Debug("Not sending WebSocket Message to the LFS because the connection is already closed")
		return
	}

	// Send the message
	if err := p.target.WriteMessage(websocket.TextMessage, msg.ToJson()); err != nil {
		logger.Warning("Failed to write message to LFS.X WebSocket: %s", err)
	}
}

// OnSourceMessage handles the proxing of a message that was received from the client WebSocket
// to the backend: Client => LFS.X
func (p *lfsxPeer) OnSourceMessage(c *websocket.Conn, messageType websocket.MessageType, data []byte) {
	logger.Debug("Received message from the client: %s", data)
	p.sourceSync.Lock()
	defer p.sourceSync.Unlock()

	// Convert the message to WebSocketData
	var wsData models.WebSocketData
	if err := json.Unmarshal(data, &wsData); err != nil {
		logger.Warning("Failed to convert message from the client: %s", err)
		return
	}
	p.notifyForUpdates(false, &wsData)

	// The other side is not yet available -> don't proxy message
	if !p.ready.Load() || p.target == nil {
		return
	}

	if err := p.target.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.Warning("Failed to write message to the LFS.X for user %q: %s", p.root.user.Username, err)
	}
}

// OnSourceMessage handles the proxing of a message that was received from the LFS.X backend
// client: LFS.X => this app ( => Client)
func (p *lfsxPeer) OnTargetMessage(c *websocket.Conn, messageType websocket.MessageType, data []byte) {
	logger.Debug("Received message from LFS.X: %s", data)
	p.targetSync.Lock()
	defer p.targetSync.Unlock()

	// Convert the message to WebSocketData
	var wsData models.WebSocketData
	if err := json.Unmarshal(data, &wsData); err != nil {
		logger.Warning("Failed to convert message from the LFS.X: %s", err)
		return
	}
	p.notifyForUpdates(true, &wsData)

	// The other side is not yet available -> don't proxy message
	if !p.ready.Load() || p.source == nil {
		return
	}

	if err := p.source.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.Warning("Failed to write message to the client for user %q: %s", p.root.user.Username, err)
	}
}
