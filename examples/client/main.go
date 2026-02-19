// Example client demonstrating how to connect to the Coin Futures WebSocket service
// using the Centrifuge Go SDK.
//
// Usage:
//
//	go run main.go -token <your_jwt_token> -endpoint ws://localhost:8009/connection
//
// Prerequisites:
//
//	go get github.com/centrifugal/centrifuge-go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/centrifugal/centrifuge-go"
)

func main() {
	token := flag.String("token", "", "JWT token for authentication")
	endpoint := flag.String("endpoint", "ws://localhost:8009/connection", "WebSocket endpoint")
	ajaibID := flag.String("ajaib-id", "130010505", "Ajaib user ID for channel subscription")
	flag.Parse()

	if *token == "" {
		log.Fatal("token is required. Use -token flag to provide JWT token")
	}

	// Create Centrifuge client with WebSocket transport
	opts := centrifuge.DefaultConfig()
	opts.Token = *token

	client := centrifuge.New(*endpoint, opts, centrifuge.JSON())

	// Set up event handlers
	client.OnConnecting(func(ctx context.Context, e centrifuge.ConnectingEvent) {
		log.Printf("Connecting to server...")
	})

	client.OnConnected(func(ctx context.Context, e centrifuge.ConnectedEvent) {
		log.Printf("Connected to server (client_id: %s)", e.ClientID)
	})

	client.OnDisconnected(func(ctx context.Context, e centrifuge.DisconnectedEvent) {
		log.Printf("Disconnected from server (code: %d, reason: %s)", e.Code, e.Reason)
	})

	// Connect to server
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Subscribe to margin channel
	marginChannel := "user:" + *ajaibID + ":margin"
	marginSub, err := client.NewSubscription(marginChannel)
	if err != nil {
		log.Fatalf("Failed to create margin subscription: %v", err)
	}

	setupSubscriptionHandlers(marginSub, "margin")
	if err := marginSub.Subscribe(); err != nil {
		log.Fatalf("Failed to subscribe to margin channel: %v", err)
	}
	log.Printf("Subscribed to margin channel: %s", marginChannel)

	// Subscribe to position channel
	positionChannel := "user:" + *ajaibID + ":position"
	positionSub, err := client.NewSubscription(positionChannel)
	if err != nil {
		log.Fatalf("Failed to create position subscription: %v", err)
	}

	setupSubscriptionHandlers(positionSub, "position")
	if err := positionSub.Subscribe(); err != nil {
		log.Fatalf("Failed to subscribe to position channel: %v", err)
	}
	log.Printf("Subscribed to position channel: %s", positionChannel)

	// Wait for interrupt signal
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)
	<-interruptChan

	log.Println("Shutting down...")
}

// setupSubscriptionHandlers configures event handlers for a subscription
func setupSubscriptionHandlers(sub *centrifuge.Subscription, channelType string) {
	sub.OnSubscribing(func(ctx context.Context, e centrifuge.SubscribingEvent) {
		log.Printf("[%s] Subscribing...", channelType)
	})

	sub.OnSubscribed(func(ctx context.Context, e centrifuge.SubscribedEvent) {
		log.Printf("[%s] Subscribed successfully (recovered: %v)", channelType, e.Recovered)
	})

	sub.OnUnsubscribed(func(ctx context.Context, e centrifuge.UnsubscribedEvent) {
		log.Printf("[%s] Unsubscribed (code: %d, reason: %s)", channelType, e.Code, e.Reason)
	})

	sub.OnPublication(func(ctx context.Context, e centrifuge.PublicationEvent) {
		handlePublication(channelType, e)
	})

	sub.OnError(func(ctx context.Context, e centrifuge.SubscriptionErrorEvent) {
		log.Printf("[%s] Subscription error: %v", channelType, e.Error)
	})
}

// handlePublication processes incoming publications
func handlePublication(channelType string, e centrifuge.PublicationEvent) {
	// Parse the publication data based on channel type
	switch channelType {
	case "margin":
		handleMarginPublication(e)
	case "position":
		handlePositionPublication(e)
	default:
		log.Printf("[%s] Received publication: %s", channelType, string(e.Data))
	}
}

// MarginPublication represents a margin update publication
type MarginPublication struct {
	CFXUserID          string  `json:"cfx_user_id"`
	Asset              string  `json:"asset"`
	MarginBalance      *string `json:"margin_balance,omitempty"`
	OrderMargin        *string `json:"order_margin,omitempty"`
	UnrealizedPnl      *string `json:"unrealized_pnl,omitempty"`
	WalletBalance      *string `json:"wallet_balance,omitempty"`
	MarginRatio        *string `json:"margin_ratio,omitempty"`
	WithdrawableMargin *string `json:"withdrawable_margin,omitempty"`
	UpdatedAt          int64   `json:"updated_at"`
}

// PositionPublication represents a position update publication
type PositionPublication struct {
	CFXUserID        string  `json:"cfx_user_id"`
	Symbol           string  `json:"symbol"`
	Size             *string `json:"size,omitempty"`
	Leverage         *string `json:"leverage,omitempty"`
	EntryPrice       *string `json:"entry_price,omitempty"`
	MarkPrice        *string `json:"mark_price,omitempty"`
	LiquidationPrice *string `json:"liquidation_price,omitempty"`
	UnrealisedPnl    *string `json:"unrealised_pnl,omitempty"`
	UpdatedAt        int64   `json:"updated_at"`
}

func handleMarginPublication(e centrifuge.PublicationEvent) {
	var pub MarginPublication
	if err := json.Unmarshal(e.Data, &pub); err != nil {
		log.Printf("[margin] Failed to parse publication: %v", err)
		log.Printf("[margin] Raw data: %s", string(e.Data))
		return
	}

	log.Printf("[margin] Update received - CFX UserID: %s, Asset: %s", pub.CFXUserID, pub.Asset)
	if pub.MarginBalance != nil {
		log.Printf("[margin]   Margin Balance: %s", *pub.MarginBalance)
	}
	if pub.WalletBalance != nil {
		log.Printf("[margin]   Wallet Balance: %s", *pub.WalletBalance)
	}
	if pub.UnrealizedPnl != nil {
		log.Printf("[margin]   Unrealized PnL: %s", *pub.UnrealizedPnl)
	}
}

func handlePositionPublication(e centrifuge.PublicationEvent) {
	var pub PositionPublication
	if err := json.Unmarshal(e.Data, &pub); err != nil {
		log.Printf("[position] Failed to parse publication: %v", err)
		log.Printf("[position] Raw data: %s", string(e.Data))
		return
	}

	log.Printf("[position] Update received - CFX UserID: %s, Symbol: %s", pub.CFXUserID, pub.Symbol)
	if pub.Size != nil {
		log.Printf("[position]   Size: %s", *pub.Size)
	}
	if pub.Leverage != nil {
		log.Printf("[position]   Leverage: %s", *pub.Leverage)
	}
	if pub.EntryPrice != nil {
		log.Printf("[position]   Entry Price: %s", *pub.EntryPrice)
	}
	if pub.MarkPrice != nil {
		log.Printf("[position]   Mark Price: %s", *pub.MarkPrice)
	}
	if pub.UnrealisedPnl != nil {
		log.Printf("[position]   Unrealized PnL: %s", *pub.UnrealisedPnl)
	}
}
