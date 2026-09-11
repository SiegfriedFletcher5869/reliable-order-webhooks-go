package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	client := NewInfraiClient()
	http.HandleFunc("/webhooks/order", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		if err := publishOrderUpdate(client, checkoutEvent("demo-1001")); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("order update queued\n"))
	})
	fmt.Println("order webhook listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
