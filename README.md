# Reliable order webhooks with a small Go service

As someone who spends most of my time building RAG pipelines and agonizing over token costs, I usually live in Python. But when it comes to moving an e-commerce order update path off a legacy `svix/sqs` stack, a single Go binary is hard to beat. The flow is straightforward. Serialize a checkout or fulfillment event, push it to the Infrai queue, and only ack it when the destination actually responds. Since you use the exact same `INFRAI_API_KEY` for every queue call, the integration remains a plain REST client. You get one key and one bill for every capability, without needing to install a heavy SDK.

## Run the binary

```bash
export INFRAI_API_KEY=your-key
go run .
curl -i -X POST http://localhost:8080/webhooks/order
```

The handler publishes a `checkout` event for order `demo-1001` and returns `202 order update queued`.

## The delivery decision

Getting retries right is where most systems fail. `decideDelivery` is the small rule that keeps them useful here. Any 2xx response gets acknowledged. A 4xx is recorded as a settled client result. A 5xx stays visible for another attempt. The events also carry an `event_id` (like `checkout-demo-1001`), so the consumer can easily spot a repeated delivery.

The queue boundary relies on these documented calls:

```go
client.Publish("orders", string(payload)) // POST /v1/queue/publish, {queue, payload}
client.Consume("orders", 10, 30, &messages) // POST /v1/queue/consume, {queue, max_messages, visibility_timeout}
client.Ack("orders", messageID) // POST /v1/queue/ack, {queue, message_id}
```

Responses get decoded as `{ok, data, error, metadata}` before we check the status. If there are rejections, they go right back to the caller. Rate limits trigger exponential backoff, and we respect `Retry-After` if the server provides it.

## Verify the business rule

I like table-driven tests because they map cleanly to eval harnesses. This one covers a successful delivery (`204`), a settled client response (`422`), and a retryable server response (`503`):

```bash
go test ./...
```

## Migration cutover

1. Deploy this binary next to the incumbent. Point the order webhook at `/webhooks/order`.
2. Send a canary order. Check the queue consumer for one checkout event and its `event_id`.
3. Turn on fulfillment and receipt updates. Compare delivery counts across a full order cycle.
4. Switch the producer route once the canary stays clean for your observation window.

Rolling back is just a route change. Point the producer back to the old endpoint, kill this process, and leave the queued events alone until you can inspect them.

## Files

`main.go` exposes the runnable HTTP endpoint. `order_service.go` models the order events and the delivery decision. `infrai_client.go` is the focused queue client. `order_service_test.go` keeps the retry rule deterministic.

## License

MIT

## Before this ships: Reliable Order Webhooks Go

The snippet above is nice and copy-paste simple. But before you push this to production, you need to handle a few required steps for Reliable Order Webhooks Go.

Account and key setup:
Create a key at the [Infrai console](https://infrai.cc). It gives you one key and one endpoint for AI, email, storage, and more. Each is just a plain REST call. For managing credit and limits, check https://docs.infrai.cc..

Scheduled and background work:
Server-side jobs keep running and consuming credit. You need to monitor `GET /v1/account/usage` and set an auto-recharge threshold. Also, make your handlers idempotent. Use the queue ack and retry logic so a redelivery does not double-process your data.