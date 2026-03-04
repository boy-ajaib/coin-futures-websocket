package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"coin-futures-websocket/internal/types"

	"github.com/centrifugal/centrifuge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransformer is a mock implementation of the Transformer interface
type mockTransformer struct {
	transformMarginFunc   func([]byte, string, string) ([]byte, error)
	transformPositionFunc func([]byte, string, string) ([]byte, error)
}

func (m *mockTransformer) TransformUserMargin(data []byte, cfxUserID string, quotePreference string) ([]byte, error) {
	if m.transformMarginFunc != nil {
		return m.transformMarginFunc(data, cfxUserID, quotePreference)
	}
	// Default: return data unchanged
	return data, nil
}

func (m *mockTransformer) TransformUserPosition(data []byte, cfxUserID string, quotePreference string) ([]byte, error) {
	if m.transformPositionFunc != nil {
		return m.transformPositionFunc(data, cfxUserID, quotePreference)
	}
	// Default: return data unchanged
	return data, nil
}

// TestNewBroadcaster tests creating a new broadcaster
func TestNewBroadcaster(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	assert.NotNil(t, broadcaster)
	assert.Equal(t, node, broadcaster.node)
	assert.Equal(t, transformer, broadcaster.transformer)
	assert.Equal(t, logger, broadcaster.logger)
	assert.NotNil(t, broadcaster.activeUsers)
}

// TestRegisterSubscription tests registering a subscription
func TestRegisterSubscription(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Register a subscription
	broadcaster.RegisterSubscription("cfx_123", "ajaib_456", "USD")

	// Verify it's registered
	user, ok := broadcaster.getSubscribedUser("cfx_123")
	assert.True(t, ok)
	assert.Equal(t, "ajaib_456", user.ajaibID)
	assert.Equal(t, "USD", user.quotePreference)
}

// TestUnregisterSubscription tests unregistering a subscription
func TestUnregisterSubscription(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Register then unregister
	broadcaster.RegisterSubscription("cfx_123", "ajaib_456", "USD")
	broadcaster.UnregisterSubscription("cfx_123")

	// Verify it's unregistered
	_, ok := broadcaster.getSubscribedUser("cfx_123")
	assert.False(t, ok)
}

// TestHandleUserMargin tests handling user margin messages
func TestHandleUserMargin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Register a subscription
	broadcaster.RegisterSubscription("cfx_123", "ajaib_456", "USD")

	// Create a user margin message
	margin := types.UserMargin{
		Timestamp:     1234567890,
		CFXUserID:     "cfx_123",
		Asset:         "USDT",
		MarginBalance: 1000.0,
	}
	data, err := json.Marshal(margin)
	require.NoError(t, err)

	// Handle the message
	err = broadcaster.handleUserMargin(data)
	assert.NoError(t, err)
}

// TestHandleUserPosition tests handling user position messages
func TestHandleUserPosition(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Register a subscription
	broadcaster.RegisterSubscription("cfx_123", "ajaib_456", "USD")

	// Create a user position message
	position := types.UserPosition{
		Timestamp: 1234567890,
		CFXUserID: "cfx_123",
		Symbol:    "BTCUSDT",
		Size:      1.5,
	}
	data, err := json.Marshal(position)
	require.NoError(t, err)

	// Handle the message
	err = broadcaster.handleUserPosition(data)
	assert.NoError(t, err)
}

// TestHandleUserMarginNoSubscriber tests handling messages when no subscribers exist
func TestHandleUserMarginNoSubscriber(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Create a user margin message with no subscribers
	margin := types.UserMargin{
		Timestamp:     1234567890,
		CFXUserID:     "cfx_999", // No subscription for this user
		Asset:         "USDT",
		MarginBalance: 1000.0,
	}
	data, err := json.Marshal(margin)
	require.NoError(t, err)

	// Handle the message - should not error
	err = broadcaster.handleUserMargin(data)
	assert.NoError(t, err)
}

// TestHandleUserPositionNoSubscriber tests handling position messages when no subscribers exist
func TestHandleUserPositionNoSubscriber(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Create a user position message with no subscribers
	position := types.UserPosition{
		Timestamp: 1234567890,
		CFXUserID: "cfx_999", // No subscription for this user
		Symbol:    "BTCUSDT",
		Size:      1.5,
	}
	data, err := json.Marshal(position)
	require.NoError(t, err)

	// Handle the message - should not error
	err = broadcaster.handleUserPosition(data)
	assert.NoError(t, err)
}

// TestHandleUserMarginWithTransformer tests that the transformer is called
func TestHandleUserMarginWithTransformer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()

	// Give the node a moment to start
	time.Sleep(10 * time.Millisecond)
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformerCalled := false
	transformer := &mockTransformer{
		transformMarginFunc: func(data []byte, cfxUserID string, quotePreference string) ([]byte, error) {
			transformerCalled = true
			assert.Equal(t, "cfx_123", cfxUserID)
			assert.Equal(t, "USD", quotePreference)
			return data, nil
		},
	}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Register a subscription
	broadcaster.RegisterSubscription("cfx_123", "ajaib_456", "USD")

	// Create a user margin message
	margin := types.UserMargin{
		Timestamp:     1234567890,
		CFXUserID:     "cfx_123",
		Asset:         "USDT",
		MarginBalance: 1000.0,
	}
	data, err := json.Marshal(margin)
	require.NoError(t, err)

	// Handle the message
	err = broadcaster.handleUserMargin(data)
	assert.NoError(t, err)
	assert.True(t, transformerCalled, "Transformer should have been called")
}

// TestHandleUserPositionWithTransformer tests that the transformer is called for positions
func TestHandleUserPositionWithTransformer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()

	// Give the node a moment to start
	time.Sleep(10 * time.Millisecond)
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformerCalled := false
	transformer := &mockTransformer{
		transformPositionFunc: func(data []byte, cfxUserID string, quotePreference string) ([]byte, error) {
			transformerCalled = true
			assert.Equal(t, "cfx_123", cfxUserID)
			assert.Equal(t, "USD", quotePreference)
			return data, nil
		},
	}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Register a subscription
	broadcaster.RegisterSubscription("cfx_123", "ajaib_456", "USD")

	// Create a user position message
	position := types.UserPosition{
		Timestamp: 1234567890,
		CFXUserID: "cfx_123",
		Symbol:    "BTCUSDT",
		Size:      1.5,
	}
	data, err := json.Marshal(position)
	require.NoError(t, err)

	// Handle the message
	err = broadcaster.handleUserPosition(data)
	assert.NoError(t, err)
	assert.True(t, transformerCalled, "Transformer should have been called")
}

// TestHandleUserMarginInvalidJSON tests handling messages with invalid JSON
func TestHandleUserMarginInvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Register a subscription
	broadcaster.RegisterSubscription("cfx_123", "ajaib_456", "USD")

	// Invalid JSON
	err = broadcaster.handleUserMargin([]byte("invalid json"))
	assert.Error(t, err)
}

// TestHandleUserPositionInvalidJSON tests handling position messages with invalid JSON
func TestHandleUserPositionInvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Register a subscription
	broadcaster.RegisterSubscription("cfx_123", "ajaib_456", "USD")

	// Invalid JSON
	err = broadcaster.handleUserPosition([]byte("invalid json"))
	assert.Error(t, err)
}

// TestHandleMessage tests the HandleMessage router
func TestHandleMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Register a subscription
	broadcaster.RegisterSubscription("cfx_123", "ajaib_456", "USD")

	t.Run("handle UserMargin topic", func(t *testing.T) {
		margin := types.UserMargin{
			Timestamp:     1234567890,
			CFXUserID:     "cfx_123",
			Asset:         "USDT",
			MarginBalance: 1000.0,
		}
		data, _ := json.Marshal(margin)

		err := broadcaster.HandleMessage(types.TopicUserMargin, []byte("key"), data)
		assert.NoError(t, err)
	})

	t.Run("handle UserPosition topic", func(t *testing.T) {
		position := types.UserPosition{
			Timestamp: 1234567890,
			CFXUserID: "cfx_123",
			Symbol:    "BTCUSDT",
			Size:      1.5,
		}
		data, _ := json.Marshal(position)

		err := broadcaster.HandleMessage(types.TopicUserPosition, []byte("key"), data)
		assert.NoError(t, err)
	})

	t.Run("handle unknown topic", func(t *testing.T) {
		err := broadcaster.HandleMessage("unknown.topic", []byte("key"), []byte("data"))
		assert.NoError(t, err) // Unknown topics are ignored, not errored
	})
}

// TestGetSubscribedUser tests retrieving subscribed users
func TestGetSubscribedUser(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Test non-existent user
	user, ok := broadcaster.getSubscribedUser("cfx_999")
	assert.False(t, ok)
	assert.Empty(t, user.ajaibID)

	// Test existing user
	broadcaster.RegisterSubscription("cfx_123", "ajaib_456", "USD")
	user, ok = broadcaster.getSubscribedUser("cfx_123")
	assert.True(t, ok)
	assert.Equal(t, "ajaib_456", user.ajaibID)
	assert.Equal(t, "USD", user.quotePreference)
}

// TestConcurrentSubscriptionTests tests concurrent access to subscriptions
func TestConcurrentSubscriptionTests(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelNone,
	})
	require.NoError(t, err)

	// Run the node in background
	go func() {
		_ = node.Run()
	}()
	defer func() { _ = node.Shutdown(context.Background()) }()

	transformer := &mockTransformer{}
	broadcaster := NewBroadcaster(node, transformer, logger)

	// Run concurrent operations
	done := make(chan bool)

	// Concurrent registrations
	for i := 0; i < 10; i++ {
		go func(index int) {
			cfxID := string(rune('a' + index))
			broadcaster.RegisterSubscription(cfxID, "ajaib_456", "USD")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all subscriptions were registered
	assert.Equal(t, 10, len(broadcaster.activeUsers))
}
