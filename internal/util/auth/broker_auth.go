package auth

import (
	"encoding/json"
	"fmt"
	"time"

	"coin-futures-websocket/internal/util/crypto"

	"github.com/google/uuid"
)

// BrokerAuthenticator handles broker authentication for CFX WebSocket
type BrokerAuthenticator struct {
	keyPair *crypto.Ed25519KeyPair
	keyID   string
}

// AuthParams represents the inner params object for authentication
type AuthParams struct {
	RequestID string `json:"request_id"`
	Timestamp int64  `json:"timestamp"`
	Method    string `json:"method"`
}

// AuthRequest represents the signed authentication request envelope
type AuthRequest struct {
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"`
	Params    string `json:"params"`
}

// NewBrokerAuthenticator creates a new broker authenticator
func NewBrokerAuthenticator(privateKeyPath string, keyID int) (*BrokerAuthenticator, error) {
	keyPair, err := crypto.LoadEd25519KeyFromFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	return &BrokerAuthenticator{
		keyPair: keyPair,
		keyID:   fmt.Sprintf("%d", keyID),
	}, nil
}

// CreateAuthRequest creates a signed authentication request for broker/auth
func (ba *BrokerAuthenticator) CreateAuthRequest() ([]byte, error) {
	params := AuthParams{
		RequestID: uuid.New().String(),
		Timestamp: time.Now().UnixMilli(),
		Method:    "broker/auth",
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	signature := ba.keyPair.Sign(paramsJSON)

	request := AuthRequest{
		KeyID:     ba.keyID,
		Signature: signature,
		Params:    string(paramsJSON),
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	return requestJSON, nil
}

// GetKeyID returns the key ID used for authentication
func (ba *BrokerAuthenticator) GetKeyID() string {
	return ba.keyID
}
