package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type VerificationMailInput struct {
	To          string
	Subject     string
	VerifyURL   string
	ExpiresIn   time.Duration
	ProductName string
}

type Sender interface {
	SendVerificationEmail(ctx context.Context, input VerificationMailInput) error
}

type ResendSender struct {
	apiKey string
	from   string
	client *http.Client
}

func NewResendSender(apiKey, from string) *ResendSender {
	return &ResendSender{
		apiKey: strings.TrimSpace(apiKey),
		from:   strings.TrimSpace(from),
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *ResendSender) SendVerificationEmail(ctx context.Context, input VerificationMailInput) error {
	if s.apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not configured")
	}

	hours := int(input.ExpiresIn.Hours())
	if hours <= 0 {
		hours = 24
	}

	product := input.ProductName
	if product == "" {
		product = "NaraLabs"
	}

	subject := input.Subject
	if subject == "" {
		subject = fmt.Sprintf("Verify your %s email", product)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html><body style="font-family:Arial,sans-serif;color:#171717;line-height:1.5">
  <h2>Verify your email</h2>
  <p>Thanks for signing up for %s. Click the button below to verify your email address.</p>
  <p><a href="%s" style="display:inline-block;background:#4A148C;color:#fff;padding:12px 20px;border-radius:8px;text-decoration:none;font-weight:600">Verify email</a></p>
  <p style="color:#737373;font-size:14px">This link expires in %d hours. If you did not create an account, you can ignore this email.</p>
  <p style="color:#737373;font-size:12px">Or copy this link:<br>%s</p>
</body></html>`, product, input.VerifyURL, hours, input.VerifyURL)

	payload := map[string]any{
		"from":    s.from,
		"to":      []string{input.To},
		"subject": subject,
		"html":    html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send resend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	return nil
}

type LogSender struct{}

func (LogSender) SendVerificationEmail(_ context.Context, input VerificationMailInput) error {
	fmt.Printf("[email] verification link for %s: %s\n", input.To, input.VerifyURL)
	return nil
}
