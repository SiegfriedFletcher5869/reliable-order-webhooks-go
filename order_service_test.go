package main

import "testing"

func TestDecideDelivery(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   DeliveryDecision
	}{
		{"delivered", 204, DeliveryDecision{Ack: true}},
		{"client rejection is settled", 422, DeliveryDecision{Ack: true}},
		{"server failure retries", 503, DeliveryDecision{Retry: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decideDelivery(tt.status); got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
