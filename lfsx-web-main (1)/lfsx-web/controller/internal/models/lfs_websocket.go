package models

import (
	"encoding/json"
	"math/rand"
	"time"

	"gitea.hama.de/LFS/go-logger"
)

// WebSocketData represents a (JSON) struct that is send over the WebSocket.
// It does wrap a list of messages while providing additional informations
// like an ID for simple request / response
type WebSocketData struct {

	// 20 bit long unique ID to identify the message
	ID int `json:"id"`

	// Used for basic "request/response" mechanism: contains the ID of the responding data
	ResponseTo int `json:"responseTo"`

	// Messages containing the "real" data
	Messages []WebSocketMessage `json:"messages"`
}

func NewWebSocketData(responseTo int, messages ...WebSocketMessage) WebSocketData {

	// Generate Random ID
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := r.Intn(1048576-0) + 0

	return WebSocketData{
		ID:         id,
		ResponseTo: responseTo,
		Messages:   messages,
	}
}

func (d WebSocketData) ToJson() []byte {
	rtc, err := json.Marshal(d)
	if err != nil {
		logger.Warning("Failed to marshal WebSocketData: %s", err)
		return []byte("{}")
	} else {
		return rtc
	}
}

// WebSocketMessage provides a list of available messages that can be send
// and received via the WebSocket.
// Note that this struct should only ever contain a single  "sub" struct that
// should have the name as given in Type.
type WebSocketMessage struct {

	// Required Field that should contain the unique type of the send message
	Type string `json:"type"`

	// Choose on of the following objects
	LoginRequest *LoginRequest `json:"loginRequest,omitempty"`
}

// LoginRequest is send from the Kubernetes controller to automatically login to the LFS
// from the provided credentials in the Web interface
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Db       string `json:"db"`
}

const LoginRequestKey = "LoginRequest"

func NewLoginRequest(username string, password string, db string) WebSocketMessage {
	return WebSocketMessage{
		Type: LoginRequestKey,
		LoginRequest: &LoginRequest{
			Username: username,
			Password: password,
			Db:       db,
		},
	}
}
