package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"coin-futures-websocket/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCfxUserMapper is a mock implementation of CfxUserMapper
type mockCfxUserMapper struct {
	cfxUserID string
	err       error
}

func (m *mockCfxUserMapper) GetCfxUserID(ctx context.Context, ajaibID int64) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.cfxUserID, nil
}

// mockUserPreferenceProvider is a mock implementation of UserPreferenceProvider
type mockUserPreferenceProvider struct {
	preference string
	err        error
}

func (m *mockUserPreferenceProvider) GetQuotePreference(ctx context.Context, ajaibID string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.preference, nil
}

// mockKafkaBroadcaster is a mock implementation of KafkaBroadcaster
type mockKafkaBroadcaster struct {
	registered   map[string]string // cfxUserID -> ajaibID
	unregistered []string          // cfxUserID
}

func newMockKafkaBroadcaster() *mockKafkaBroadcaster {
	return &mockKafkaBroadcaster{
		registered: make(map[string]string),
	}
}

func (m *mockKafkaBroadcaster) RegisterSubscription(cfxUserID, ajaibID, quotePreference string) {
	m.registered[cfxUserID] = ajaibID
}

func (m *mockKafkaBroadcaster) UnregisterSubscription(cfxUserID string) {
	m.unregistered = append(m.unregistered, cfxUserID)
	delete(m.registered, cfxUserID)
}

// TestNewCentrifugeServer tests creating a new Centrifuge server
func TestNewCentrifugeServer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.CentrifugeConfiguration{
		NodeName:  "test-node",
		Namespace: "test-ns",
		LogLevel:  "info",
	}

	server := NewCentrifugeServer(cfg, logger)

	assert.NotNil(t, server)
	assert.NotNil(t, server.node)
	assert.NotNil(t, server.wsHandler)
	assert.Equal(t, cfg, server.config)
	assert.Equal(t, logger, server.logger)
}

// TestSetDependencies tests setting the server dependencies
func TestSetDependencies(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.CentrifugeConfiguration{
		NodeName:  "test-node",
		Namespace: "test-ns",
		LogLevel:  "info",
	}

	server := NewCentrifugeServer(cfg, logger)

	// Set mock dependencies
	mapper := &mockCfxUserMapper{cfxUserID: "test_cfx_123"}
	prefProvider := &mockUserPreferenceProvider{preference: "USD"}
	broadcaster := newMockKafkaBroadcaster()

	server.SetCfxUserMapper(mapper)
	server.SetUserPreferenceProvider(prefProvider)
	server.SetBroadcaster(broadcaster)

	assert.NotNil(t, server.cfxUserMapper)
	assert.NotNil(t, server.userPrefProvider)
	assert.NotNil(t, server.broadcaster)
}

// TestClientInfoSerialization tests ClientInfo serialization
func TestClientInfoSerialization(t *testing.T) {
	info := ClientInfo{
		AjaibID:         "12345",
		CfxUserID:       "cfx_123",
		QuotePreference: "USD",
		ConnectedAt:     1234567890,
	}

	// Test JSON marshaling
	data, err := json.Marshal(info)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test JSON unmarshaling
	var unmarshaled ClientInfo
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, info.AjaibID, unmarshaled.AjaibID)
	assert.Equal(t, info.CfxUserID, unmarshaled.CfxUserID)
	assert.Equal(t, info.QuotePreference, unmarshaled.QuotePreference)
	assert.Equal(t, info.ConnectedAt, unmarshaled.ConnectedAt)
}

// TestClientInfoGetters tests ClientInfo getter methods
func TestClientInfoGetters(t *testing.T) {
	info := ClientInfo{
		AjaibID:         "12345",
		CfxUserID:       "cfx_123",
		QuotePreference: "USD",
		ConnectedAt:     1234567890,
	}

	assert.Equal(t, "12345", info.GetAjaibID())
	assert.Equal(t, "cfx_123", info.GetCfxUserID())
	assert.Equal(t, "USD", info.GetQuotePreference())
	assert.Equal(t, int64(1234567890), info.GetConnectedAt())
}

// TestErrorCodes tests that error codes are within expected ranges
func TestErrorCodes(t *testing.T) {
	// Client errors (4000-4499) - non-terminal
	assert.GreaterOrEqual(t, uint32(CodeBadRequest), uint32(4000))
	assert.Less(t, uint32(CodeBadRequest), uint32(4500))

	assert.GreaterOrEqual(t, uint32(CodeChannelNotFound), uint32(4000))
	assert.Less(t, uint32(CodeChannelNotFound), uint32(4500))

	// Authorization errors (4100-4199)
	assert.GreaterOrEqual(t, uint32(CodeUnauthorized), uint32(4100))
	assert.Less(t, uint32(CodeUnauthorized), uint32(4200))

	// Server errors (4500-4999) - terminal
	assert.GreaterOrEqual(t, uint32(CodeInternalError), uint32(4500))
	assert.Less(t, uint32(CodeInternalError), uint32(5000))

	assert.GreaterOrEqual(t, uint32(CodeCfxUserResolution), uint32(4500))
	assert.Less(t, uint32(CodeCfxUserResolution), uint32(5000))
}

// TestNewError tests creating Centrifuge errors
func TestNewError(t *testing.T) {
	err := NewError(CodeBadRequest, "bad request")
	assert.NotNil(t, err)
	assert.Equal(t, uint32(4000), err.Code)
	assert.Equal(t, "bad request", err.Message)
}

// TestDisconnectReasons tests disconnect reason messages
func TestDisconnectReasons(t *testing.T) {
	assert.NotEmpty(t, DisconnectReasons.Unauthorized())
	assert.NotEmpty(t, DisconnectReasons.ChannelNotFound())
	assert.NotEmpty(t, DisconnectReasons.CfxUserResolutionError())
	assert.NotEmpty(t, DisconnectReasons.UserPreferenceError())

	assert.Contains(t, DisconnectReasons.Unauthorized(), "unauthorized")
	assert.Contains(t, DisconnectReasons.ChannelNotFound(), "channel not found")
	assert.Contains(t, DisconnectReasons.CfxUserResolutionError(), "service unavailable")
}

// TestGetClientInfo tests extracting client info from a Centrifuge client
func TestGetClientInfo(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.CentrifugeConfiguration{
		NodeName:  "test-node",
		Namespace: "test-ns",
		LogLevel:  "info",
	}

	server := NewCentrifugeServer(cfg, logger)
	_ = server // Use server to avoid unused variable warning

	// Test with empty info bytes
	// Since we can't create a real Client without connecting, we'll test the logic indirectly
	// by testing the ClientInfo serialization/deserialization

	// Test with valid info
	clientInfo := ClientInfo{
		AjaibID:         "12345",
		CfxUserID:       "cfx_123",
		QuotePreference: "USD",
		ConnectedAt:     1234567890,
	}
	infoData, _ := json.Marshal(clientInfo)

	// Deserialize to verify the structure
	var parsedInfo ClientInfo
	err := json.Unmarshal(infoData, &parsedInfo)
	require.NoError(t, err)
	assert.Equal(t, "12345", parsedInfo.AjaibID)
	assert.Equal(t, "cfx_123", parsedInfo.CfxUserID)
	assert.Equal(t, "USD", parsedInfo.QuotePreference)
	assert.Equal(t, int64(1234567890), parsedInfo.ConnectedAt)
}

// TestResolveCfxUserID tests resolving CFX user ID from Ajaib ID
func TestResolveCfxUserID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.CentrifugeConfiguration{
		NodeName:  "test-node",
		Namespace: "test-ns",
		LogLevel:  "info",
	}

	server := NewCentrifugeServer(cfg, logger)

	// Test without mapper
	cfxID, err := server.resolveCfxUserID(context.Background(), "12345")
	assert.Error(t, err)
	assert.Empty(t, cfxID)

	// Test with mapper
	mapper := &mockCfxUserMapper{cfxUserID: "cfx_123"}
	server.SetCfxUserMapper(mapper)

	cfxID, err = server.resolveCfxUserID(context.Background(), "12345")
	assert.NoError(t, err)
	assert.Equal(t, "cfx_123", cfxID)

	// Test with invalid Ajaib ID
	cfxID, err = server.resolveCfxUserID(context.Background(), "invalid")
	assert.Error(t, err)
	assert.Empty(t, cfxID)

	// Test with mapper error
	errorMapper := &mockCfxUserMapper{err: assert.AnError}
	server.SetCfxUserMapper(errorMapper)

	cfxID, err = server.resolveCfxUserID(context.Background(), "12345")
	assert.Error(t, err)
	assert.Empty(t, cfxID)
}

// TestResolveQuotePreference tests fetching user quote preference
func TestResolveQuotePreference(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.CentrifugeConfiguration{
		NodeName:  "test-node",
		Namespace: "test-ns",
		LogLevel:  "info",
	}

	server := NewCentrifugeServer(cfg, logger)

	// Test without provider
	pref, err := server.resolveQuotePreference(context.Background(), "12345")
	assert.NoError(t, err) // No error when provider is nil
	assert.Empty(t, pref)

	// Test with provider
	prefProvider := &mockUserPreferenceProvider{preference: "USD"}
	server.SetUserPreferenceProvider(prefProvider)

	pref, err = server.resolveQuotePreference(context.Background(), "12345")
	assert.NoError(t, err)
	assert.Equal(t, "USD", pref)

	// Test with provider error
	errorProvider := &mockUserPreferenceProvider{err: assert.AnError}
	server.SetUserPreferenceProvider(errorProvider)

	pref, err = server.resolveQuotePreference(context.Background(), "12345")
	assert.Error(t, err)
	assert.Empty(t, pref)
}

// TestGetClientCount tests getting the client count
func TestGetClientCount(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.CentrifugeConfiguration{
		NodeName:  "test-node",
		Namespace: "test-ns",
		LogLevel:  "info",
	}

	server := NewCentrifugeServer(cfg, logger)

	// Initially should be 0
	count := server.GetClientCount()
	assert.Equal(t, 0, count)

	// Verify server is properly initialized
	assert.NotNil(t, server)
	assert.NotNil(t, server.node)
}
