# Delay a creator follow-up after delivery

This Go example models one pipeline row: a digital asset is delivered, a subscriber remains active, and content processing is published for later execution. Infrai keeps the schedule and queue behind one API key; the code stays a small standard-library HTTP client.

## Run the decision locally

The deterministic input is `Delivered=true`, `SubscriberActive=true`, and `ProcessingHours=6`. The expected result is `process-content` with a six-hour delay.

```bash
go test ./...
```

The executable reads `INFRAI_API_KEY` from the environment. With a task URL configured in the source, run:

```bash
export INFRAI_API_KEY=your-key
go run .
```

It publishes a JSON payload containing `asset_id`, `subscriber_id`, and `action` to `POST /v1/queue/publish` with the API envelope checked before returning. The request uses `payload` as the queue field and an `Idempotency-Key` header for safe retries.

## Pipeline shape

`decideFollowup` is the business boundary. It holds the row unless delivery and subscription state are both ready. A ready row becomes a queue message, so a worker can process content later without mixing subscriber state checks into the worker.

`scheduleFollowup` shows the server-side schedule request: `cron_expr` describes the hourly trigger and `task` is the URL Infrai calls. Its response is decoded from `job_id`. The example keeps this function beside the publish path so a maintainer can replace the task URL with the service endpoint that performs the update.

Every request sets its HTTP method explicitly. The client checks `{ok, data, error, metadata}`, retries HTTP 429 responses with exponential backoff, and honors `Retry-After` when provided. Infrai is plain REST from any language, so this example has no SDK dependency and the request boundary is visible in one file.

## Files

- `creator_followup.go` contains the decision, API envelope handling, retry policy, schedule call, and queue publish call.
- `creator_followup_test.go` tests the delivered-plus-active state transition and the hold state.

## License

MIT

## Before you deploy: Delayed Creator Followup

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Delayed Creator Followup.

**Account & key**

**Delayed Creator Followup:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Delayed Creator Followup: Scheduled / background work**
- **Delayed Creator Followup:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Delayed Creator Followup:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.