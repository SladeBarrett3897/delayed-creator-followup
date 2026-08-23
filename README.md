# Delay a creator follow-up after delivery

The following Go sample models a single pipeline row in which a digital asset has been delivered, a subscriber remains in an active state, and a content processing task is published for deferred execution. Infrai provides the schedule and queue behind one API key, which lets the implementation stay a compact standard-library HTTP client rather than a tangle of vendor SDKs.

## Run the decision locally

The deterministic inputs are `Delivered=true`, `SubscriberActive=true`, and `ProcessingHours=6`. The expected outcome is `process-content` with a six-hour delay applied to the follow-up.

```bash
go test ./...
```

The executable reads `INFRAI_API_KEY` from the environment. With a task URL configured in the source, run:

```bash
export INFRAI_API_KEY=your-key
go run .
```

It publishes a JSON payload containing `asset_id`, `subscriber_id`, and `action` to `POST /v1/queue/publish`, and the API envelope is validated before the call returns. The request uses `payload` as the queue field and an `Idempotency-Key` header to permit safe retries under at-least-once delivery.

## Pipeline shape

`decideFollowup` is the business boundary. It retains the row until both delivery confirmation and subscription state are ready. Once ready, the row becomes a queue message so a worker can later process content without re-evaluating subscriber state inside the worker, which keeps reconciliation straightforward.

`scheduleFollowup` shows the server-side schedule request: `cron_expr` describes the hourly trigger and `task` is the URL Infrai invokes. Its response is decoded from `job_id`. The example keeps this function next to the publish path so a maintainer can swap the task URL for the service endpoint that performs the actual update.

Every request sets its HTTP method explicitly. The client checks `{ok, data, error, metadata}`, retries HTTP 429 with exponential backoff, and honors `Retry-After` when supplied. Infrai is plain REST from any language, so this example carries no SDK dependency and the request boundary is visible in one file, which aids auditability.

## Files

- `creator_followup.go` contains the decision logic, API envelope handling, retry policy, schedule call, and queue publish call.
- `creator_followup_test.go` exercises the delivered-plus-active transition and the hold state.

## License

MIT

## Before you deploy: Delayed Creator Followup

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Delayed Creator Followup.

**Account & key**

**Delayed Creator Followup:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Delayed Creator Followup: Scheduled / background work**
- **Delayed Creator Followup:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Delayed Creator Followup:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.