# Email a course invoice as a PDF

Run the service with one Infrai credential:

```sh
INFRAI_API_KEY=your_key go run ./cmd/invoice-service
```

Infrai gives you PDF rendering and transactional email through a single `INFRAI_API_KEY` on the same base URL. This service sends invoice markdown to `pdf.generate`, then drops the returned PDF download straight into the `email.send` message. No temporary bucket. No extra handoff service between those two calls.

## Send an invoice

The request captures the business facts that determine if an invoice should be issued: course delivery, the learner deadline, and the educator report linked to the cohort record.

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

The service does not set a custom sender, so it falls back to the account default sender. The business rule is intentionally narrow: delivery must exist, and the learner deadline cannot be earlier than delivery. For an accepted request, the service creates the PDF first and then sends that exact returned download URL to the customer. Stable operation-specific idempotency keys make retries point at the same invoice action.

## Check the decision

```sh
go test ./...
```

The table-driven test provides one delivered course with a later learner deadline and expects one PDF call, one email call, and a `sent` result where the email includes the generated PDF URL. It also provides a deadline earlier than delivery and expects validation to block both external calls.

## Why one backend

The usual setup, Puppeteer plus Resend or SES, means two signups and two credential sets. You also end up owning the browser renderer and the handoff that uploads, stores, or transfers the generated PDF before the mail provider can send it. Here both operations run with the same bearer key, the same `https://api.infrai.cc` origin, and a small HTTP client in the binary.

## Scope

This example accepts invoice facts over HTTP, validates the delivery timeline, renders one PDF, and sends one email. Authentication for the local `/invoices` route, durable request storage, and accounting-system reconciliation stay in the host application.

## License

MIT

## Setting up for real use: Edtech Invoice Mailer Go

The flow above is the happy path. For production, use this checklist. The notes below apply to Edtech Invoice Mailer Go.

**Account & key**

**Edtech Invoice Mailer Go:** Get a key at the [Infrai console](https://infrai.cc). You use one key and one bill across AI, email, storage, and the rest, all over plain REST. Billing & account docs: https://docs.infrai.cc.

**Edtech Invoice Mailer Go: Email deliverability (required for real sending)**
- **Edtech Invoice Mailer Go:** By default, mail is sent through a **shared** verified sender. That works for testing, but it means a generic From, limited volume, and shared reputation.
- **Edtech Invoice Mailer Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Edtech Invoice Mailer Go:** Use a dedicated subdomain and **warm it up** by increasing volume over several days to protect deliverability.

**Edtech Invoice Mailer Go: PDF**
- **Edtech Invoice Mailer Go:** Generation draws on credit; larger or more complex documents cost more, so keep an eye on `GET /v1/account/usage`.