//go:build integration

package integration_test

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"coin-futures-websocket/config"
	"coin-futures-websocket/internal/websocket/server"

	centrifugeclient "github.com/centrifugal/centrifuge-go"
)

// buildTestToken crafts a minimal unsigned JWT with {"sub": ajaibID} as payload.
// internal/auth/jwt.go does not verify signatures — it only base64-decodes the payload
// and reads the "sub" claim — so a hand-built token works fine.
func buildTestToken(ajaibID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"` + ajaibID + `"}`))
	return header + "." + payload + ".fake-signature"
}

// ─── Mock types ────────────────────────────────────────────────────────────────

// mockCfxUserMapper implements server.CfxUserMapper.
type mockCfxUserMapper struct {
	cfxUserID string
	err       error
}

func (m *mockCfxUserMapper) GetCfxUserID(_ context.Context, _ int64) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.cfxUserID, nil
}

// mockUserPreferenceProvider implements server.UserPreferenceProvider.
type mockUserPreferenceProvider struct {
	preference string
	err        error
}

func (m *mockUserPreferenceProvider) GetQuotePreference(_ context.Context, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.preference, nil
}

// mockKafkaBroadcaster implements server.KafkaBroadcaster and records calls.
type mockKafkaBroadcaster struct {
	mu           sync.Mutex
	registered   map[string]string // cfxUserID → ajaibID
	unregistered []string          // cfxUserIDs passed to UnregisterSubscription
}

func newMockKafkaBroadcaster() *mockKafkaBroadcaster {
	return &mockKafkaBroadcaster{registered: make(map[string]string)}
}

func (m *mockKafkaBroadcaster) RegisterSubscription(cfxUserID, ajaibID, _ string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registered[cfxUserID] = ajaibID
}

func (m *mockKafkaBroadcaster) UnregisterSubscription(cfxUserID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unregistered = append(m.unregistered, cfxUserID)
	delete(m.registered, cfxUserID)
}

func (m *mockKafkaBroadcaster) isRegistered(cfxUserID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.registered[cfxUserID]
	return ok
}

func (m *mockKafkaBroadcaster) wasUnregistered(cfxUserID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.unregistered {
		if id == cfxUserID {
			return true
		}
	}
	return false
}

// ─── Server factory ────────────────────────────────────────────────────────────

// testServerHandle groups the objects needed to interact with a running test server.
type testServerHandle struct {
	URL      string // ws:// URL of the /connection endpoint
	wsServer *server.CentrifugeServer
}

// startTestServer creates and starts a CentrifugeServer backed by mocks, registers it
// with an httptest.Server on a random port, and returns the handles.
// Server shutdown is registered with t.Cleanup.
func startTestServer(
	t *testing.T,
	mapper server.CfxUserMapper,
	prefProvider server.UserPreferenceProvider,
	broadcaster server.KafkaBroadcaster,
) *testServerHandle {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := &config.CentrifugeConfiguration{
		NodeName: "integration-test-node",
		LogLevel: "error",
	}

	wsServer := server.NewCentrifugeServer(cfg, logger)
	wsServer.SetCfxUserMapper(mapper)
	wsServer.SetUserPreferenceProvider(prefProvider)
	wsServer.SetBroadcaster(broadcaster)

	if err := wsServer.Start(); err != nil {
		t.Fatalf("startTestServer: Start() failed: %v", err)
	}

	// Give the Centrifuge node a moment to fully initialise before accepting connections.
	time.Sleep(20 * time.Millisecond)

	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsServer.ServeHTTP(w, r)
	}))

	t.Cleanup(func() {
		httpSrv.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = wsServer.Shutdown(ctx)
	})

	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")

	return &testServerHandle{
		URL:      wsURL,
		wsServer: wsServer,
	}
}

// ─── Wait helper ───────────────────────────────────────────────────────────────

// waitFor polls condition() every 10 ms until it returns true or timeout elapses.
// Calls t.Fatal on timeout.
func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("waitFor: condition not met within %s", timeout)
}

// ─── Client factory ────────────────────────────────────────────────────────────

// connectClient creates a centrifuge-go client, connects it, and waits for the
// OnConnected event. Registers t.Cleanup to close the client on test teardown.
// The endpoint should be the ws:// URL returned by startTestServer.
func connectClient(t *testing.T, endpoint, token string) *centrifugeclient.Client {
	t.Helper()

	connected := make(chan struct{})
	disconnected := make(chan centrifugeclient.DisconnectedEvent, 1)

	client := centrifugeclient.NewJsonClient(endpoint+"/connection", centrifugeclient.Config{
		Token:             token,
		MinReconnectDelay: 30 * time.Second, // avoid noisy reconnect loops in tests
		MaxReconnectDelay: 60 * time.Second,
	})

	client.OnConnected(func(e centrifugeclient.ConnectedEvent) {
		select {
		case <-connected: // already closed
		default:
			close(connected)
		}
	})

	client.OnDisconnected(func(e centrifugeclient.DisconnectedEvent) {
		select {
		case disconnected <- e:
		default:
		}
	})

	if err := client.Connect(); err != nil {
		t.Fatalf("connectClient: Connect() returned error: %v", err)
	}

	t.Cleanup(func() { client.Close() })

	select {
	case <-connected:
		return client
	case e := <-disconnected:
		t.Fatalf("connectClient: disconnected before connected (code=%d reason=%q)", e.Code, e.Reason)
	case <-time.After(5 * time.Second):
		t.Fatalf("connectClient: timeout waiting for connection")
	}

	return client // unreachable, but satisfies the compiler
}
