package invoice

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeIssuer struct {
	pdfCalls   int
	emailCalls int
	markdown   string
	html       string
	pdfKey     string
	emailKey   string
}

func (f *fakeIssuer) GeneratePDF(_ context.Context, markdown, key string) (string, error) {
	f.pdfCalls++
	f.markdown = markdown
	f.pdfKey = key
	return "https://downloads.example/inv-1042.pdf", nil
}

func (f *fakeIssuer) SendEmail(_ context.Context, _, _, body, key string) (string, error) {
	f.emailCalls++
	f.html = body
	f.emailKey = key
	return "msg_1042", nil
}

func TestIssueDecision(t *testing.T) {
	delivered := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		deadline  time.Time
		wantErr   bool
		wantCalls int
	}{
		{name: "delivered course with later deadline is invoiced", deadline: delivered.AddDate(0, 0, 14), wantCalls: 1},
		{name: "deadline before delivery is rejected", deadline: delivered.AddDate(0, 0, -1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeIssuer{}
			sender := NewReceiptSender(fake)
			result, err := sender.Issue(context.Background(), InvoiceRequest{
				InvoiceID: "INV-1042", CustomerName: "Mina Chen", CustomerEmail: "mina@example.com",
				CourseTitle: "Practical Data Literacy", CourseDeliveredAt: delivered, LearnerDeadline: tt.deadline,
				EducatorReportName: "September cohort completion", Amount: "480.00", Currency: "USD",
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Issue() error = %v, wantErr %v", err, tt.wantErr)
			}
			if fake.pdfCalls != tt.wantCalls || fake.emailCalls != tt.wantCalls {
				t.Fatalf("calls = PDF %d email %d, want %d each", fake.pdfCalls, fake.emailCalls, tt.wantCalls)
			}
			if tt.wantErr {
				return
			}
			if result.Status != "sent" || !strings.Contains(fake.html, "https://downloads.example/inv-1042.pdf") {
				t.Fatalf("result = %#v, email body = %q", result, fake.html)
			}
			if fake.pdfKey == fake.emailKey || fake.pdfKey == "" || fake.emailKey == "" {
				t.Fatalf("write keys must be stable and operation-specific")
			}
		})
	}
}
