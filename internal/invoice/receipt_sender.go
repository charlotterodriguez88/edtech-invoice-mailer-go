package invoice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"time"
)

type Issuer interface {
	GeneratePDF(context.Context, string, string) (string, error)
	SendEmail(context.Context, string, string, string, string) (string, error)
}

type InvoiceRequest struct {
	InvoiceID          string    `json:"invoice_id"`
	CustomerName       string    `json:"customer_name"`
	CustomerEmail      string    `json:"customer_email"`
	CourseTitle        string    `json:"course_title"`
	CourseDeliveredAt  time.Time `json:"course_delivered_at"`
	LearnerDeadline    time.Time `json:"learner_deadline"`
	EducatorReportName string    `json:"educator_report_name"`
	Amount             string    `json:"amount"`
	Currency           string    `json:"currency"`
}

type InvoiceResult struct {
	InvoiceID string `json:"invoice_id"`
	PDFURL    string `json:"pdf_url"`
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

type ReceiptSender struct{ api Issuer }

func NewReceiptSender(api Issuer) *ReceiptSender { return &ReceiptSender{api: api} }

func (s *ReceiptSender) Issue(ctx context.Context, in InvoiceRequest) (InvoiceResult, error) {
	if err := validate(in); err != nil {
		return InvoiceResult{}, err
	}

	key := stableKey(in.InvoiceID)
	pdfURL, err := s.api.GeneratePDF(ctx, invoiceMarkdown(in), key+"-pdf")
	if err != nil {
		return InvoiceResult{}, fmt.Errorf("generate invoice PDF: %w", err)
	}
	messageID, err := s.api.SendEmail(ctx, in.CustomerEmail, "Invoice "+in.InvoiceID+" for "+in.CourseTitle, invoiceEmail(in, pdfURL), key+"-email")
	if err != nil {
		return InvoiceResult{}, fmt.Errorf("email invoice: %w", err)
	}
	return InvoiceResult{InvoiceID: in.InvoiceID, PDFURL: pdfURL, MessageID: messageID, Status: "sent"}, nil
}

func validate(in InvoiceRequest) error {
	if in.InvoiceID == "" || in.CustomerName == "" || in.CustomerEmail == "" || in.CourseTitle == "" || in.Amount == "" || in.Currency == "" {
		return &ValidationError{Message: "invoice, customer, course, amount, and currency are required"}
	}
	if in.CourseDeliveredAt.IsZero() {
		return &ValidationError{Message: "course delivery must be recorded before invoicing"}
	}
	if in.LearnerDeadline.IsZero() || in.LearnerDeadline.Before(in.CourseDeliveredAt) {
		return &ValidationError{Message: "learner deadline must be on or after course delivery"}
	}
	if in.EducatorReportName == "" {
		return &ValidationError{Message: "educator report name is required"}
	}
	return nil
}

func invoiceMarkdown(in InvoiceRequest) string {
	return fmt.Sprintf("# Invoice %s\n\nCustomer: %s\n\nCourse: %s\n\nDelivered: %s\n\nLearner deadline: %s\n\nEducator report: %s\n\nTotal: %s %s\n", in.InvoiceID, in.CustomerName, in.CourseTitle, in.CourseDeliveredAt.Format("2006-01-02"), in.LearnerDeadline.Format("2006-01-02"), in.EducatorReportName, in.Amount, in.Currency)
}

func invoiceEmail(in InvoiceRequest, pdfURL string) string {
	return fmt.Sprintf("<p>Hello %s,</p><p>Your invoice for %s is ready.</p><p><a href=\"%s\">Download invoice PDF</a></p><p>Learner deadline: %s.<br>Educator report: %s.</p>", html.EscapeString(in.CustomerName), html.EscapeString(in.CourseTitle), html.EscapeString(pdfURL), in.LearnerDeadline.Format("2006-01-02"), html.EscapeString(in.EducatorReportName))
}

func stableKey(invoiceID string) string {
	sum := sha256.Sum256([]byte("edtech-invoice:" + invoiceID))
	return hex.EncodeToString(sum[:16])
}

var _ error = (*ValidationError)(nil)
