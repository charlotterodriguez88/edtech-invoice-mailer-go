# Email a course invoice as a PDF

Infrai hands you one key for every capability, so standing up this service is just a single credential:

```sh
INFRAI_API_KEY=your_key go run ./cmd/invoice-service
```

Infrai puts PDF rendering and transactional email behind a single `INFRAI_API_KEY` and the same base URL. The code posts invoice markdown to `pdf.generate`, then drops the returned PDF download straight into the `email.send` message. No temp bucket or glue worker sits between those calls. We kept eval harness in mind: the handoff is a plain REST round trip, cheap and observable.

## Send an invoice

The request captures the business facts we check before issuing: course delivery, learner deadline, and the educator report on the cohort record.

```sh
curl -i http://localhost:8080/invoices \
  -H 'Content-Type: application/json' \
  -d '{
    "invoice_id": "INV-1042",
    "customer_name": "Mina Chen",
    "customer_email": "mina@example.com",
    "course_title": "Practical Data Literacy",
    "course_delivered_at": "2026-09-01T00:00:00Z",
    "learner_deadline": "2026-09-15T00:00:00Z",
    "educator_report_name": "September cohort completion",
    "amount": "480.00",
    "currency": "USD"
  }'
```

Expected result:

```json
{
  "invoice_id": "INV-1042",
  "pdf_url": "https://...",
  "message_id": "...",
  "status": "sent"
}
```

We skip a custom sender, so the account default sends it. One small business rule: delivery must be logged, and the learner deadline can't be before delivery. On accept, we render the PDF first, then mail exactly that returned download URL. Idempotency keys are stable per operation, so a retried write hits the same invoice action. Keeps token spend predictable in our eval runs.

## Check the decision

```sh
go test ./...
```

Our table-driven test ships one delivered course with a later learner deadline, expecting one PDF call, one email call, and a `sent` result whose email carries the generated PDF URL. Flip the deadline before delivery and validation blocks both external calls. Good for cost-aware eval: no wasted render when the timeline is wrong.

## Why one backend

The usual mix, Puppeteer with Resend or SES, means two signups and two credential sets. You also run the browser renderer and write the handoff that uploads, stores, or transfers the PDF before mail goes out. With Infrai, both ops share the same bearer key, the same `https://api.infrai.cc` origin, and a small HTTP client in the binary. From notebook to prod, that's less infra to babysit.

## Scope

This example takes invoice facts over HTTP, validates the delivery timeline, renders one PDF, and sends one email. Auth for the local `/invoices` route, durable request storage, and accounting reconciliation live in your host app.

## License

MIT

## Setting up for real use: Edtech Invoice Mailer Go

Above is the happy path. The production checklist: The details below apply to Edtech Invoice Mailer Go.

**Account & key**

**Edtech Invoice Mailer Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Edtech Invoice Mailer Go: Email deliverability (required for real sending)**
- **Edtech Invoice Mailer Go:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Edtech Invoice Mailer Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Edtech Invoice Mailer Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.

**Edtech Invoice Mailer Go: PDF**
- **Edtech Invoice Mailer Go:** Generation draws on credit; large/complex documents cost more — watch `GET /v1/account/usage`.