package main

import (
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

	client := centrifuge.NewJsonClient(*endpoint, centrifuge.Config{
		Token:             *token,
		MinReconnectDelay: 500 * time.Millisecond,
		MaxReconnectDelay: 10 * time.Second,
	})

	client.OnConnecting(func(e centrifuge.ConnectingEvent) {
		log.Printf("Connecting to server... (code: %d, reason: %s)", e.Code, e.Reason)
	})

	client.OnConnected(func(e centrifuge.ConnectedEvent) {
		log.Printf("Connected to server (client_id: %s)", e.ClientID)
	})

	client.OnDisconnected(func(e centrifuge.DisconnectedEvent) {
		log.Printf("Disconnected from server (code: %d, reason: %s)", e.Code, e.Reason)
	})

	client.OnError(func(e centrifuge.ErrorEvent) {
		log.Printf("Client error: %v", e.Error)
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

	// Wait for interrupt signal
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt)
	<-interruptChan

	log.Println("Shutting down...")
}

// setupSubscriptionHandlers configures event handlers for a subscription
func setupSubscriptionHandlers(sub *centrifuge.Subscription, channelType string) {
	sub.OnSubscribing(func(e centrifuge.SubscribingEvent) {
		log.Printf("[%s] Subscribing... (code: %d, reason: %s)", channelType, e.Code, e.Reason)
	})

	sub.OnSubscribed(func(e centrifuge.SubscribedEvent) {
		log.Printf("[%s] Subscribed successfully (recovered: %v)", channelType, e.Recovered)
	})

	sub.OnUnsubscribed(func(e centrifuge.UnsubscribedEvent) {
		log.Printf("[%s] Unsubscribed (code: %d, reason: %s)", channelType, e.Code, e.Reason)
	})

	sub.OnPublication(func(e centrifuge.PublicationEvent) {
		handlePublication(channelType, e)
	})

	sub.OnError(func(e centrifuge.SubscriptionErrorEvent) {
		log.Printf("[%s] Subscription error: %v", channelType, e.Error)
	})
}

// handlePublication processes incoming publications
func handlePublication(channelType string, e centrifuge.PublicationEvent) {
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
	Timestamp          int64   `json:"timestamp"`
	CFXUserID          string  `json:"cfx_user_id"`
	Asset              string  `json:"asset"`
	TotalPositionValue float64 `json:"total_position_value"`
	MarginBalance      float64 `json:"margin_balance"`
	OrderMargin        float64 `json:"order_margin"`
	EffectiveLeverage  float64 `json:"effective_leverage"`
	MaintenanceMargin  float64 `json:"maintenance_margin"`
	UnrealizedPnl      float64 `json:"unrealized_pnl"`
	AvailableMargin    float64 `json:"available_margin"`
	WalletBalance      float64 `json:"wallet_balance"`
	MarginRatio        float64 `json:"margin_ratio"`
	WithdrawableMargin float64 `json:"withdrawable_margin"`
}

// PositionPublication represents a position update publication
type PositionPublication struct {
	Timestamp                int64   `json:"timestamp"`
	CFXUserID                string  `json:"cfx_user_id"`
	Symbol                   string  `json:"symbol"`
	Size                     float64 `json:"size"`
	Value                    float64 `json:"value"`
	Leverage                 float64 `json:"leverage"`
	EntryPrice               float64 `json:"entry_price"`
	MarkPrice                float64 `json:"mark_price"`
	LiquidationPrice         float64 `json:"liquidation_price"`
	MaintenanceMargin        float64 `json:"maintenance_margin"`
	RealisedPnl              float64 `json:"realised_pnl"`
	UnrealisedPnl            float64 `json:"unrealised_pnl"`
	DeleveragePercentile     float64 `json:"deleverage_percentile"`
	RiskLimit                float64 `json:"risk_limit"`
	OpenOrderBuyCost         float64 `json:"open_order_buy_cost"`
	OpenOrderSellCost        float64 `json:"open_order_sell_cost"`
	InitialMarginRequirement float64 `json:"initial_margin_requirement"`
	UpdatedTime              int64   `json:"updated_time"`
	OpenOrderBuyQuantity     float64 `json:"open_order_buy_quantity"`
	OpenOrderSellQuantity    float64 `json:"open_order_sell_quantity"`
	OrderMargin              float64 `json:"order_margin"`
}

func handleMarginPublication(e centrifuge.PublicationEvent) {
	var pub MarginPublication
	if err := json.Unmarshal(e.Data, &pub); err != nil {
		log.Printf("[margin] Failed to parse publication: %v — raw: %s", err, string(e.Data))
		return
	}

	log.Printf("[margin] %s", string(e.Data))
}

func handlePositionPublication(e centrifuge.PublicationEvent) {
	var pub PositionPublication
	if err := json.Unmarshal(e.Data, &pub); err != nil {
		log.Printf("[position] Failed to parse publication: %v — raw: %s", err, string(e.Data))
		return
	}

	log.Printf("[position] %s", string(e.Data))
}
