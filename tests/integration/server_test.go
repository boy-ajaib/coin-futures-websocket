//go:build integration

package integration_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	centrifugeclient "github.com/centrifugal/centrifuge-go"
)

const (
	testAjaibID  = "130010505"
	testCfxID    = "cfx-test-user-001"
	testPref     = "USDT"
	dialTimeout  = 5 * time.Second
	eventTimeout = 3 * time.Second
)

// ─── Connection flow ───────────────────────────────────────────────────────────

func TestConnect_ValidJWT(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	// connectClient asserts that Connected is reached before returning.
	_ = connectClient(t, srv.URL, buildTestToken(testAjaibID))
}

func TestConnect_MissingToken(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	disconnected := make(chan centrifugeclient.DisconnectedEvent, 1)
	client := centrifugeclient.NewJsonClient(srv.URL+"/connection", centrifugeclient.Config{
		// No token
		MinReconnectDelay: 30 * time.Second,
		MaxReconnectDelay: 60 * time.Second,
	})
	client.OnDisconnected(func(e centrifugeclient.DisconnectedEvent) {
		select {
		case disconnected <- e:
		default:
		}
	})

	t.Cleanup(func() { client.Close() })
	require.NoError(t, client.Connect())

	select {
	case e := <-disconnected:
		assert.EqualValues(t, 4100, e.Code, "expected CodeUnauthorized (4100)")
	case <-time.After(eventTimeout):
		t.Fatal("timeout: expected disconnect for missing token")
	}
}

func TestConnect_InvalidJWT(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	disconnected := make(chan centrifugeclient.DisconnectedEvent, 1)
	client := centrifugeclient.NewJsonClient(srv.URL+"/connection", centrifugeclient.Config{
		Token:             "this.is.not.a.valid.jwt.at.all",
		MinReconnectDelay: 30 * time.Second,
		MaxReconnectDelay: 60 * time.Second,
	})
	client.OnDisconnected(func(e centrifugeclient.DisconnectedEvent) {
		select {
		case disconnected <- e:
		default:
		}
	})

	t.Cleanup(func() { client.Close() })
	require.NoError(t, client.Connect())

	select {
	case e := <-disconnected:
		assert.EqualValues(t, 4100, e.Code, "expected CodeUnauthorized (4100)")
	case <-time.After(eventTimeout):
		t.Fatal("timeout: expected disconnect for invalid JWT")
	}
}

func TestConnect_CfxMapperFailure(t *testing.T) {
	mapper := &mockCfxUserMapper{err: fmt.Errorf("cfx adapter down")}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	disconnected := make(chan centrifugeclient.DisconnectedEvent, 1)
	client := centrifugeclient.NewJsonClient(srv.URL+"/connection", centrifugeclient.Config{
		Token:             buildTestToken(testAjaibID),
		MinReconnectDelay: 30 * time.Second,
		MaxReconnectDelay: 60 * time.Second,
	})
	client.OnDisconnected(func(e centrifugeclient.DisconnectedEvent) {
		select {
		case disconnected <- e:
		default:
		}
	})

	t.Cleanup(func() { client.Close() })
	require.NoError(t, client.Connect())

	select {
	case e := <-disconnected:
		assert.EqualValues(t, 4501, e.Code, "expected CodeCfxUserResolution (4501)")
	case <-time.After(eventTimeout):
		t.Fatal("timeout: expected disconnect for CFX mapper failure")
	}
}

func TestConnect_PrefProviderFailure(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{err: fmt.Errorf("coin-setting unavailable")}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	disconnected := make(chan centrifugeclient.DisconnectedEvent, 1)
	client := centrifugeclient.NewJsonClient(srv.URL+"/connection", centrifugeclient.Config{
		Token:             buildTestToken(testAjaibID),
		MinReconnectDelay: 30 * time.Second,
		MaxReconnectDelay: 60 * time.Second,
	})
	client.OnDisconnected(func(e centrifugeclient.DisconnectedEvent) {
		select {
		case disconnected <- e:
		default:
		}
	})

	t.Cleanup(func() { client.Close() })
	require.NoError(t, client.Connect())

	select {
	case e := <-disconnected:
		assert.EqualValues(t, 4502, e.Code, "expected CodeUserPreference (4502)")
	case <-time.After(eventTimeout):
		t.Fatal("timeout: expected disconnect for preference provider failure")
	}
}

// ─── Subscription flow ─────────────────────────────────────────────────────────

func TestSubscribe_ValidChannel(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	client := connectClient(t, srv.URL, buildTestToken(testAjaibID))

	channel := "user:" + testAjaibID + ":margin"
	sub, err := client.NewSubscription(channel)
	require.NoError(t, err)

	subscribed := make(chan struct{})
	sub.OnSubscribed(func(e centrifugeclient.SubscribedEvent) { close(subscribed) })
	require.NoError(t, sub.Subscribe())

	select {
	case <-subscribed:
	case <-time.After(eventTimeout):
		t.Fatal("timeout: expected successful subscription")
	}

	// Broadcaster must have been notified.
	waitFor(t, eventTimeout, func() bool { return bc.isRegistered(testCfxID) })
}

func TestSubscribe_InvalidChannelFormat(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	client := connectClient(t, srv.URL, buildTestToken(testAjaibID))

	sub, err := client.NewSubscription("invalid-channel")
	require.NoError(t, err)

	subErr := make(chan centrifugeclient.SubscriptionErrorEvent, 1)
	sub.OnError(func(e centrifugeclient.SubscriptionErrorEvent) {
		select {
		case subErr <- e:
		default:
		}
	})
	require.NoError(t, sub.Subscribe())

	select {
	case e := <-subErr:
		var serverErr *centrifugeclient.Error
		require.True(t, errors.As(e.Error, &serverErr), "expected *centrifuge.Error, got %T: %v", e.Error, e.Error)
		assert.EqualValues(t, 4001, serverErr.Code, "expected CodeChannelNotFound (4001)")
	case <-time.After(eventTimeout):
		t.Fatal("timeout: expected subscription error for invalid channel")
	}
}

func TestSubscribe_WrongUser(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	// Connect as testAjaibID but subscribe to a channel belonging to a different user.
	client := connectClient(t, srv.URL, buildTestToken(testAjaibID))

	differentUserChannel := "user:999999999:margin"
	sub, err := client.NewSubscription(differentUserChannel)
	require.NoError(t, err)

	subErr := make(chan centrifugeclient.SubscriptionErrorEvent, 1)
	sub.OnError(func(e centrifugeclient.SubscriptionErrorEvent) {
		select {
		case subErr <- e:
		default:
		}
	})
	require.NoError(t, sub.Subscribe())

	select {
	case e := <-subErr:
		var serverErr *centrifugeclient.Error
		require.True(t, errors.As(e.Error, &serverErr), "expected *centrifuge.Error, got %T: %v", e.Error, e.Error)
		assert.EqualValues(t, 4001, serverErr.Code, "expected CodeChannelNotFound (4001)")
	case <-time.After(eventTimeout):
		t.Fatal("timeout: expected subscription error for wrong user channel")
	}
}

func TestSubscribe_MultipleChannels(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	client := connectClient(t, srv.URL, buildTestToken(testAjaibID))

	marginSubscribed := make(chan struct{})
	positionSubscribed := make(chan struct{})

	marginCh := "user:" + testAjaibID + ":margin"
	marginSub, err := client.NewSubscription(marginCh)
	require.NoError(t, err)
	marginSub.OnSubscribed(func(e centrifugeclient.SubscribedEvent) { close(marginSubscribed) })
	require.NoError(t, marginSub.Subscribe())

	positionCh := "user:" + testAjaibID + ":position"
	positionSub, err := client.NewSubscription(positionCh)
	require.NoError(t, err)
	positionSub.OnSubscribed(func(e centrifugeclient.SubscribedEvent) { close(positionSubscribed) })
	require.NoError(t, positionSub.Subscribe())

	select {
	case <-marginSubscribed:
	case <-time.After(eventTimeout):
		t.Fatal("timeout: margin channel subscription")
	}
	select {
	case <-positionSubscribed:
	case <-time.After(eventTimeout):
		t.Fatal("timeout: position channel subscription")
	}
}

// ─── Broadcast flow ────────────────────────────────────────────────────────────

func TestBroadcast_MarginMessage(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	client := connectClient(t, srv.URL, buildTestToken(testAjaibID))

	channel := "user:" + testAjaibID + ":margin"
	sub, err := client.NewSubscription(channel)
	require.NoError(t, err)

	subscribed := make(chan struct{})
	sub.OnSubscribed(func(e centrifugeclient.SubscribedEvent) { close(subscribed) })

	publications := make(chan centrifugeclient.PublicationEvent, 1)
	sub.OnPublication(func(e centrifugeclient.PublicationEvent) {
		select {
		case publications <- e:
		default:
		}
	})

	require.NoError(t, sub.Subscribe())
	select {
	case <-subscribed:
	case <-time.After(eventTimeout):
		t.Fatal("timeout waiting for subscription")
	}

	payload := []byte(`{"cfx_user_id":"` + testCfxID + `","asset":"BTC","margin_balance":"1000.0"}`)
	_, err = srv.wsServer.Node().Publish(channel, payload)
	require.NoError(t, err)

	select {
	case pub := <-publications:
		assert.Equal(t, payload, pub.Data)
	case <-time.After(eventTimeout):
		t.Fatal("timeout: expected publication on margin channel")
	}
}

func TestBroadcast_PositionMessage(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	client := connectClient(t, srv.URL, buildTestToken(testAjaibID))

	channel := "user:" + testAjaibID + ":position"
	sub, err := client.NewSubscription(channel)
	require.NoError(t, err)

	subscribed := make(chan struct{})
	sub.OnSubscribed(func(e centrifugeclient.SubscribedEvent) { close(subscribed) })

	publications := make(chan centrifugeclient.PublicationEvent, 1)
	sub.OnPublication(func(e centrifugeclient.PublicationEvent) {
		select {
		case publications <- e:
		default:
		}
	})

	require.NoError(t, sub.Subscribe())
	select {
	case <-subscribed:
	case <-time.After(eventTimeout):
		t.Fatal("timeout waiting for subscription")
	}

	payload := []byte(`{"cfx_user_id":"` + testCfxID + `","symbol":"BTCUSDT","size":"0.5"}`)
	_, err = srv.wsServer.Node().Publish(channel, payload)
	require.NoError(t, err)

	select {
	case pub := <-publications:
		assert.Equal(t, payload, pub.Data)
	case <-time.After(eventTimeout):
		t.Fatal("timeout: expected publication on position channel")
	}
}

func TestBroadcast_NoSubscriber(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	// No client subscribed — Publish should still succeed without error.
	channel := "user:" + testAjaibID + ":margin"
	payload := []byte(`{"cfx_user_id":"` + testCfxID + `","asset":"BTC"}`)
	_, err := srv.wsServer.Node().Publish(channel, payload)
	assert.NoError(t, err, "Publish with no subscriber should not return an error")
}

// ─── Disconnect flow ───────────────────────────────────────────────────────────

func TestDisconnect_CleanClose(t *testing.T) {
	mapper := &mockCfxUserMapper{cfxUserID: testCfxID}
	pref := &mockUserPreferenceProvider{preference: testPref}
	bc := newMockKafkaBroadcaster()
	srv := startTestServer(t, mapper, pref, bc)

	client := connectClient(t, srv.URL, buildTestToken(testAjaibID))

	channel := "user:" + testAjaibID + ":margin"
	sub, err := client.NewSubscription(channel)
	require.NoError(t, err)

	subscribed := make(chan struct{})
	sub.OnSubscribed(func(e centrifugeclient.SubscribedEvent) { close(subscribed) })
	require.NoError(t, sub.Subscribe())

	select {
	case <-subscribed:
	case <-time.After(eventTimeout):
		t.Fatal("timeout waiting for subscription")
	}

	// Ensure broadcaster has the registration before closing.
	waitFor(t, eventTimeout, func() bool { return bc.isRegistered(testCfxID) })

	// Close the client — the server's disconnect handler should call UnregisterSubscription.
	client.Close()

	waitFor(t, eventTimeout, func() bool { return bc.wasUnregistered(testCfxID) })
}
