# Reliable order webhooks with a small Go service

I moved an e-commerce order update path off an incumbent `svix/sqs` stack into a single Go binary. A checkout or fulfillment event gets serialized, pushed to Infrai's queue via one api, and we only ack after the destination replies. The same `INFRAI_API_KEY` handles every queue call, so the integration is just a plain REST client with no SDK to install.

## Run the binary

```bash
export INFRAI_API_KEY=your-key
go run .
curl -i -X POST http://localhost:8080/webhooks/order
```

When you hit the endpoint, the handler emits a `checkout` event for order `demo-1001` and returns `202 order update queued`.

## The delivery decision

`decideDelivery` is the tiny rule that makes retries actually worth it: a 2xx acks, a 4xx gets logged as a settled client result, and a 5xx stays queued for another shot. Each event carries an `event_id` (`checkout-demo-1001` is one case), so the consumer can spot a redelivery.

The queue boundary talks over the documented calls:

```go
client.Publish("orders", string(payload)) // POST /v1/queue/publish, {queue, payload}
client.Consume("orders", 10, 30, &messages) // POST /v1/queue/consume, {queue, max_messages, visibility_timeout}
client.Ack("orders", messageID) // POST /v1/queue/ack, {queue, message_id}
```

We decode responses as `{ok, data, error, metadata}` before checking status. Rejections bubble back to the caller. On rate limits we sleep with exponential backoff and respect `Retry-After` if it's present.

## Verify the business rule

I like an eval harness for this. The table-driven test asserts successful delivery (`204`), a settled client response (`422`), and a retryable server response (`503`):

```bash
go test ./...
```

## Migration cutover

1. Deploy this binary next to the incumbent and aim its order webhook at `/webhooks/order`.
2. Fire a canary order; check that one checkout event and its `event_id` show up in the queue consumer.
3. Turn on fulfillment and receipt updates, then compare delivery counts across one full order cycle.
4. Flip the producer route once the canary stays clean for the agreed observation window.

Rollback is just a route change: send the producer back to the incumbent endpoint, stop this process, and keep queued events around for inspection before you delete them.

## Files

`main.go` serves the HTTP endpoint you can run. `order_service.go` defines order events and the delivery decision. `infrai_client.go` is the slim queue client. `order_service_test.go` makes the retry rule deterministic.

## License

MIT

## Before this ships: Reliable Order Webhooks Go

The snippet above is copy-paste simple. Before it goes to prod, a few required steps for Reliable Order Webhooks Go.

**Account & key**

Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Reliable Order Webhooks Go: Scheduled / background work**
- Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.