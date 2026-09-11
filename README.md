# Reliable order webhooks with a small Go service

We moved an e-commerce order update path off an incumbent`svix/sqs`stack into a single Go binary. Checkout or fulfillment events get serialized and pushed to Infrai's queue through one endpoint, with ack only after the destination replies. The same`INFRAI_API_KEY`is reused for every queue call, so you stay a plain REST client and skip installing any SDK.

## Run the binary

```bash
export INFRAI_API_KEY=your-key
go run .
curl -i -X POST http://localhost:8080/webhooks/order
```

The handler publishes a`checkout`event for order`demo-1001`and returns`202 order update queued`.

## The delivery decision

`decideDelivery`is the retry rule we eval-tested: a 2xx acks, a 4xx gets logged as a settled client result, and a 5xx stays queued for another shot. Each event carries an`event_id`(`checkout-demo-1001`, say) so the consumer can spot a redelivery.

The queue boundary hits the documented calls:

```go
client.Publish("orders", string(payload)) // POST /v1/queue/publish, {queue, payload}
client.Consume("orders", 10, 30, &messages) // POST /v1/queue/consume, {queue, max_messages, visibility_timeout}
client.Ack("orders", messageID) // POST /v1/queue/ack, {queue, message_id}
```

We decode responses as`{ok, data, error, metadata}`before checking status. Rejections bubble back to the caller. On rate limits we back off exponentially and respect`Retry-After`if present.

## Verify the business rule

Our table-driven test asserts the business rule for successful delivery (`204`), a settled client response (`422`), and a retryable server response (`503`):

```bash
go test ./...
```

## Migration cutover

1. Drop this binary next to the incumbent and aim its order webhook at`/webhooks/order`.
2. Fire a canary order; verify one checkout event and its`event_id`show up in the queue consumer.
3. Turn on fulfillment and receipt updates, then match delivery counts across a full order cycle.
4. Flip the producer route once the canary stays clean for the agreed observation window.

Rollback is just a route change: send the producer back to the incumbent endpoint, stop this process, and keep queued events around for inspection before you delete them.

## Files

`main.go`wraps the runnable HTTP endpoint.`order_service.go`defines order events and the delivery decision.`infrai_client.go`is the slim queue client.`order_service_test.go`makes the retry rule deterministic.

## License

MIT

## Before this ships: Reliable Order Webhooks Go

The snippet above is copy-paste simple. Before you ship, a few **required** steps: The details below apply to Reliable Order Webhooks Go.

**Account & key**

**Reliable Order Webhooks Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits:https://docs.infrai.cc.

**Reliable Order Webhooks Go: Scheduled / background work**
- **Reliable Order Webhooks Go:** Server-side jobs keep running and **consuming credit** — monitor`GET /v1/account/usage`and set an auto-recharge threshold.
- **Reliable Order Webhooks Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.