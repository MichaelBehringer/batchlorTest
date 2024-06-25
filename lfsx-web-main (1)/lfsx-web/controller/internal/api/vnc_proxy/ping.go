package vnc

import (
	"context"
	"sync"
	"time"

	"gitea.hama.de/LFS/go-logger"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

const KeepAliveTimeout = 15 * time.Second

// ClientMgr handles the Ping Pong messages between the WebSocket clients
type ClientMgr struct {
	mux           sync.Mutex
	context       context.Context
	clients       map[*websocket.Conn]*websocket.Conn
	keepaliveTime time.Duration
}

func NewClientMgr(keepaliveTime time.Duration, ctx context.Context) *ClientMgr {
	return &ClientMgr{
		context:       ctx,
		clients:       make(map[*websocket.Conn]*websocket.Conn, 0),
		keepaliveTime: keepaliveTime,
	}
}

func (cm *ClientMgr) Add(c *websocket.Conn) {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	cm.clients[c] = c
}

func (cm *ClientMgr) Delete(c *websocket.Conn) {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	delete(cm.clients, c)
}

func (cm *ClientMgr) Run() {
	ticker := time.NewTicker(cm.keepaliveTime - (2 * time.Second))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go func() {
				cm.mux.Lock()
				for wsConn := range cm.clients {
					if err := wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
						logger.Debug("Keepalive: closing connection to %q because of send error: %s", wsConn.RemoteAddr().String(), err)

						go func(con *websocket.Conn) {
							if err := con.Close(); err != nil {
								logger.Debug("Unable to close clonnection: %s", err)
								cm.Delete(con)
							}
						}(cm.clients[wsConn])
					}
				}
				cm.mux.Unlock()
				logger.Trc("Keppalive: pinged %v clients", len(cm.clients))
			}()
		case <-cm.context.Done():
			return
		}
	}
}
