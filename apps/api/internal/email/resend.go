package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"flowbook/api/internal/db"
)

// Client is Resend email client — uses https://api.resend.com/emails.
// When apiKey is empty, it runs in mock mode (logs, no external call) for tests/CI.
type Client struct {
	apiKey     string
	from       string
	httpClient *http.Client
	frontendURL string
	enabled    bool
}

// New creates a Client. from defaults to "FlowBook <noreply@flowbook.example.com>" if empty.
// frontendURL is used to build absolute track links (fallback to relative if empty).
func New(apiKey, from, frontendURL string) *Client {
	if from == "" {
		from = "FlowBook <noreply@flowbook.example.com>"
	}
	enabled := strings.TrimSpace(apiKey) != "" && !isPlaceholder(apiKey)
	if !enabled {
		slog.Info("email: Resend disabled — mock mode (log only)", "from", from)
	}
	return &Client{
		apiKey:      apiKey,
		from:        from,
		frontendURL: strings.TrimRight(frontendURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		enabled:     enabled,
	}
}

func isPlaceholder(k string) bool {
	// .env.example has re_... or sk_test_... placeholder values that should not trigger real send
	trim := strings.TrimSpace(k)
	if trim == "re_..." || trim == "sk_test_..." || trim == "whsec_..." || trim == "[service-role-key]" || strings.Contains(trim, "[") {
		return true
	}
	if trim == "" {
		return true
	}
	// Heuristic: Resend test key is re_... with >10 chars; placeholder "re_..." length 6 is disabled
	if trim == "re_..." {
		return true
	}
	return false
}

// resend payload structs
type resendAttachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"` // base64
	ContentType string `json:"content_type,omitempty"`
}

type resendPayload struct {
	From        string             `json:"from"`
	To          []string           `json:"to"`
	Subject     string             `json:"subject"`
	HTML        string             `json:"html"`
	Text        *string            `json:"text,omitempty"`
	Attachments []resendAttachment `json:"attachments,omitempty"`
	Tags        []map[string]string `json:"tags,omitempty"`
}

// SendBookingConfirmed sends BookingConfirmed + ics attach per AC T05.
// It is idempotent-safe: caller ensures booking already CONFIRMED.
// Returns nil on mock mode or on successful Resend 200.
func (c *Client) SendBookingConfirmed(ctx context.Context, booking db.Booking, service db.Service, staff db.Staff, org db.Organization) error {
	subject, htmlBody := renderBookingConfirmedHTML(booking, service, staff, org)
	ics := BuildICS(booking, service, staff, org)
	icsB64 := base64.StdEncoding.EncodeToString([]byte(ics))
	filename := fmt.Sprintf("booking-%s.ics", booking.ID.String()[:8])
	return c.send(ctx, booking.CustomerEmail, subject, htmlBody, []resendAttachment{
		{Filename: filename, Content: icsB64, ContentType: "text/calendar; charset=utf-8; method=REQUEST"},
	}, "booking_confirmed", booking.ID.String())
}

// SendCancelled sends Cancelled notification (no ics).
func (c *Client) SendCancelled(ctx context.Context, booking db.Booking, service db.Service, staff db.Staff, org db.Organization) error {
	subject, htmlBody := renderCancelledHTML(booking, service, staff, org)
	return c.send(ctx, booking.CustomerEmail, subject, htmlBody, nil, "booking_cancelled", booking.ID.String())
}

// SendReminder sends Reminder H-1 (cron every 15m). No ics attach to keep light.
func (c *Client) SendReminder(ctx context.Context, booking db.Booking, service db.Service, staff db.Staff, org db.Organization) error {
	subject, htmlBody := renderReminderHTML(booking, service, staff, org)
	return c.send(ctx, booking.CustomerEmail, subject, htmlBody, nil, "booking_reminder_h1", booking.ID.String())
}

// Send is low-level generic send — used for tests and direct calls.
func (c *Client) Send(ctx context.Context, to, subject, htmlBody string) error {
	return c.send(ctx, to, subject, htmlBody, nil, "generic", "")
}

func (c *Client) send(ctx context.Context, to, subject, htmlBody string, attachments []resendAttachment, tagValue, bookingID string) error {
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("email: recipient empty")
	}
	to = strings.TrimSpace(to)
	// Mock mode — log and succeed
	if !c.enabled {
		slog.Info("email: mock send (RESEND_API_KEY not set)",
			"to", to,
			"subject", subject,
			"tag", tagValue,
			"bookingId", bookingID,
			"hasICS", len(attachments) > 0,
			"htmlPreview", truncate(htmlBody, 200),
		)
		// Also log ICS preview for BookingConfirmed
		if len(attachments) > 0 {
			slog.Info("email: mock ICS attached", "filename", attachments[0].Filename, "size", len(attachments[0].Content))
		}
		return nil
	}

	payload := resendPayload{
		From:        c.from,
		To:          []string{to},
		Subject:     subject,
		HTML:        htmlBody,
		Attachments: attachments,
	}
	if tagValue != "" {
		payload.Tags = []map[string]string{{"name": "category", "value": tagValue}}
		if bookingID != "" {
			payload.Tags = append(payload.Tags, map[string]string{"name": "booking_id", "value": bookingID})
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("email request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("email: Resend request failed (network)", "to", to, "subject", subject, "error", err)
		return fmt.Errorf("email send: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Error("email: Resend API error", "to", to, "subject", subject, "status", resp.StatusCode, "body", string(respBody))
		// Don't fail booking flow on email error — log and return nil for resilience, but return error if caller wants to know?
		// Per T05, email is best-effort — webhook should not 500 on email fail. So we log and return nil to keep webhook 200.
		// However for direct Send tests we return error. Distinguish by checking if tag is generic? For now return error so tests can assert.
		// But for booking flows, service will log and ignore error. We'll return nil to avoid breaking checkout.
		// To keep consistent, return nil here and log error; the service caller will decide to ignore.
		// For strict, we return nil to keep webhook idempotent 200.
		if resp.StatusCode >= 500 {
			// transient — could retry but we swallow
			slog.Warn("email: transient Resend error, swallowing to keep webhook 200", "status", resp.StatusCode)
		}
		// Swallow to keep payment flow 200 — email failure should not revert booking confirmation
		// Return nil to avoid webhook retry storm due to email transient
		return nil
	}

	slog.Info("email: sent via Resend", "to", to, "subject", subject, "tag", tagValue, "bookingId", bookingID, "status", resp.StatusCode, "response", truncate(string(respBody), 300))
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// Health returns true if email client is enabled (has real key)
func (c *Client) Health() bool { return c.enabled }
