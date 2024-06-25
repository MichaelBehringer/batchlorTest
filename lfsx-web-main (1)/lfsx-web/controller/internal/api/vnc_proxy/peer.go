package vnc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http/httputil"
	"strings"
	"sync/atomic"
	"time"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/guacamole"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/models"
	"github.com/lesismal/nbio"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

// peer represents a vnc proxy peer with a
// websocket connection the client connects to
// and a vnc backend connection that will be proxied
// to the client.
// It does also contain a http reverse proxy to talk with the LFS.X
// API and the host API
type peer struct {

	// VNC WebSocket connection of the client
	source *websocket.Conn

	// VNC backend connection the user wants to use
	target net.Conn

	// Guacamole connection the user wants to use
	targetGuacamole *net.Conn

	// The request user
	user *models.User

	// The Peer to the LFS.X Kubernetes WebSocket
	lfsxPeer *lfsxPeer

	// Weather the underlaying connection belongs to a noVNC proxy
	// or a guacamole proxy
	guacamole Guacamole

	// ready states if both connections are setup for
	// further use
	ready atomic.Bool

	// if this peer was already closed
	closed atomic.Bool

	// A reverse proxy to the LFS.X API
	lfsxAPI *httputil.ReverseProxy
	// A reverse proxy to the host API
	hostAPI *httputil.ReverseProxy

	// function that is called when this peer goes offline
	onDisconnect func(*peer, error, int)
}

// NewPeer creates an empty peer for the given user.
//
// For using this peer you have to call also the function
// "SetConnections()" after you have initialized both
// connections
func NewPeer(user *models.User, onDisconnect func(*peer, error, int)) *peer {
	return &peer{
		user:         user,
		onDisconnect: onDisconnect,
	}
}

const InternalDataOpcode = ""

var internalOpcodeIns = []byte(fmt.Sprint(len(InternalDataOpcode), ".", InternalDataOpcode))

// SetConnection is setting both connections for the peer
// and making it so ready for usage
func (p *peer) SetConnections(source *websocket.Conn, target *nbio.Conn, targetGuacamole *net.Conn, lfsx *httputil.ReverseProxy, hostAPI *httputil.ReverseProxy) {
	p.source = source
	p.target = target
	p.lfsxAPI = lfsx
	p.hostAPI = hostAPI
	p.targetGuacamole = targetGuacamole

	// Make both peers ready
	p.ready.Store(true)
}

// IsReady returns weather this peer is ready
func (p *peer) IsReady() bool {
	return p.ready.Load()
}

// ReadSource copy source stream to target connection
func (p *peer) ReadSource() error {
	if _, err := io.Copy(p.target, p.source.Conn); err != nil {
		return fmt.Errorf("copy source(%v) => target(%v) failed: %s", p.source.RemoteAddr(), p.target.RemoteAddr(), err)
	}
	return nil
}

// ReadTarget copys target stream to source connection
func (p *peer) ReadTarget() error {
	if _, err := io.Copy(p.source, p.target); err != nil {
		return fmt.Errorf("copy target(%v) => source(%v) failed: %s", p.target.RemoteAddr(), p.source.RemoteAddr(), err)
	}
	return nil
}

// Close close the websocket connection and the vnc backend connection.
//
// You can provide an error message (close reason) and a toggle from which
// side the close event came (1 = source, 2 = target)
func (p *peer) Close(err error, fromWhich int) {
	logger.Debug("Closing peer connection (from %d): %s", fromWhich, err)

	// Don't close a connection multiple times
	if p.closed.Load() {
		logger.Debug("Peer was already closed. Not closing...")
		return
	} else {
		p.closed.Store(true)
	}

	// We wait until all connections are fully initialized. In some scenarious the disconnect happens right
	// after connecting to the LFS.X or VNC connection.
	// If we would close the connection directly before the VNC handshake is finished, the Close() call would
	// have no effect or the connectino doesn't even exist already.
	// The connections would be stall because no other code piece is using that
	i := 0
	for {
		if !p.ready.Load() {
			logger.Debug("Connections are not fully loaded in close handler. Wating (%d)...", i)
			time.Sleep(400 * time.Millisecond)

			// Limit by 50
			if i > 50 {
				logger.Error("Initialization of connections before closing them timed out for user %q", p.user.Username)
				break
			}
			i++
		} else {
			break
		}
	}
	// Print a warning message because this should not happen
	if i > 2 {
		logger.Warning("Needed to wait %d milliseconds until connections could be closed", 400*i)
	}

	// Close the LFS.X peer if available
	if p.lfsxPeer != nil {
		// We don't pass the information from which the connection was closed further down.
		// This would lead to a connection race because inside the LFS.X peer it does expect the partner closed the connection.
		// But that's not the case at all!
		p.lfsxPeer.Close(err, 0)
	}

	if p.source != nil && fromWhich != 1 {
		// Close the WebSocket with a reason if one was given.
		// When we receive a nil error, the connection was closed on the server side
		if err != nil {
			var codeBytes = make([]byte, 2)
			binary.BigEndian.PutUint16(codeBytes, 1008)
			codeBytes = append(codeBytes, err.Error()...)
			p.source.WriteMessage(websocket.CloseMessage, codeBytes)
		}

		p.source.Close()
	}

	if p.target != nil && fromWhich != 2 && !p.guacamole.Used {
		logger.Debug("Closed connection to the VNC backend")
		p.target.Close()
	}

	if p.targetGuacamole != nil && fromWhich != 2 {
		logger.Debug("Closing connection to guacd")
		(*p.targetGuacamole).Close()
	}

	p.onDisconnect(p, err, fromWhich)
}

// OnSourceMessage handles the proxing of a message that was received from the WebSocket
// client: WebSocket => TCP VNC
func (p *peer) OnSourceMessage(c *websocket.Conn, messageType websocket.MessageType, data []byte) {
	if !p.ready.Load() {
		return
	}

	if p.guacamole.Used {
		if bytes.HasPrefix(data, internalOpcodeIns) {
			// messages starting with the InternalDataOpcode are never sent to guacd
			return
		}

		if _, err := p.guacamole.Writer.Write(data); err != nil {
			logger.Debug("Failed writing message to guacd: %s", err)
			return
		}
	} else {
		if _, err := p.target.Write(data); err != nil {
			logger.Warning("Failed to write message to VNC backend for user %q: %s", p.user.Username, err)
		}
	}

}

// OnSourceMessage handles the proxing of a message that was received from the VNC backend
// client: TCP VNC => WebSocket
func (p *peer) OnTargetMessage(c *nbio.Conn, data []byte) {
	if !p.ready.Load() {
		return
	}

	if err := p.source.WriteMessage(websocket.BinaryMessage, data); err != nil {
		logger.Warning("Failed to write messsage to WebSocket client %q: %s", p.user.Username, err)
	}
}

// proxyGuacamole establishes a connection to guacamole
func (p *peer) proxyGuacamole(quality string) error {
	p.guacamole.Used = true

	// Generate config
	config := guacamole.NewGuacamoleConfiguration()
	config.Protocol = "vnc"
	config.Parameters["hostname"] = "127.0.0.1"
	config.Parameters["port"] = "5910"
	config.Parameters["cursor"] = "local"
	config.Parameters["autoretry"] = "true"

	// 8 = really bad (you see each color) | 16 = For most data no difference | 24 = very good also for pictures | 32 = no difference
	colorDepth := "24"
	if quality == "medium" {
		colorDepth = "16"
	} else if quality == "low" {
		colorDepth = "8"
	}
	config.Parameters["color-depth"] = colorDepth

	// Image types: webp = best performance, looks really good | jpeg = bad performance, looks like webp | png = bad performance, but looks really good
	// Don't set a single value. When all types are supported, guacamole tries to use an compression dynamically based on how fast elements are updated
	config.ImageMimetypes = []string{"image/webp", "image/jpeg", "image/png"}

	config.AudioMimetypes = []string{"audio/L16", "rate=44100", "channels=2"}

	// Connect to guacd
	logger.Debug("Connecting to guacd")
	stream := guacamole.NewStream(*p.targetGuacamole, KeepAliveTimeout)
	if err := stream.Handshake(config); err != nil {
		return err
	}
	p.guacamole.Stream = stream

	// Proxy from WebSocket -> Guacd
	go func() {
		// Create tunnel
		tunnel := guacamole.NewSimpleTunnel(stream)
		writer := tunnel.AcquireWriter()
		p.guacamole.Writer = writer
		reader := tunnel.AcquireReader()

		// Cleanup
		defer tunnel.ReleaseWriter()
		defer tunnel.ReleaseReader()

		// Proxy from Guacd -> WebSocket
		buf := bytes.NewBuffer(make([]byte, 0, guacamole.MaxGuacMessage*2))

		for {
			ins, err := reader.ReadSome()
			if err != nil {
				logger.Debug("Error reading from guacd: %s", err)

				// Guacd closed the connection. So call the close event
				if err.Error() == "EOF" {
					p.Close(err, -1)
				}

				return
			}

			if bytes.HasPrefix(ins, internalOpcodeIns) {
				// messages starting with the InternalDataOpcode are never sent to the websocket
				continue
			}

			if _, err = buf.Write(ins); err != nil {
				logger.Debug("Failed to buffer guacd to ws: %s", err)
				return
			}

			// if the buffer has more data in it or we've reached the max buffer size, send the data and reset
			if !reader.Available() || buf.Len() >= guacamole.MaxGuacMessage {
				if err = p.source.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
					logger.Debug("Failed sending message to ws: %s", err)
					// Client closed connection=
					if strings.Contains(err.Error(), "closed network") {
						logger.Debug("Use of a closed network connection of guacamole VNVC client -> terminate peer")
						p.Close(err, -1)
					}
					return
				}
				buf.Reset()
			}
		}

	}()

	return nil
}
