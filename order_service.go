package main

import (
	"encoding/json"
	"fmt"
)

type OrderEvent struct {
	EventID string `json:"event_id"`
	OrderID string `json:"order_id"`
	Kind    string `json:"kind"`
}

type DeliveryDecision struct {
	Publish bool
	Ack     bool
	Retry   bool
}

func decideDelivery(status int) DeliveryDecision {
	if status >= 200 && status < 300 {
		return DeliveryDecision{Ack: true}
	}
	if status >= 400 && status < 500 {
		return DeliveryDecision{Ack: true}
	}
	return DeliveryDecision{Retry: true}
}

func publishOrderUpdate(client *InfraiClient, event OrderEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return client.Publish("orders", string(payload))
}

func checkoutEvent(orderID string) OrderEvent {
	return OrderEvent{EventID: "checkout-" + orderID, OrderID: orderID, Kind: "checkout"}
}
func fulfillmentEvent(orderID string) OrderEvent {
	return OrderEvent{EventID: "fulfillment-" + orderID, OrderID: orderID, Kind: "fulfillment"}
}

func (e OrderEvent) String() string { return fmt.Sprintf("%s:%s", e.OrderID, e.Kind) }
