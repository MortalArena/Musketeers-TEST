package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MortalArena/Musketeers/pkg/capability"
	"github.com/MortalArena/Musketeers/pkg/policy"
)

type GmailCapability struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewGmailCapability(token string) *GmailCapability {
	return &GmailCapability{BaseURL: "https://www.googleapis.com/gmail/v1", Token: token, HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

func (c *GmailCapability) Name() string { return "gmail" }

func (c *GmailCapability) Execute(ctx context.Context, principal policy.Principal, cmd capability.Command) (*capability.Result, error) {
	switch v := cmd.(type) {
	case SendEmailCommand:
		return c.sendEmail(ctx, v)
	case ListEmailsCommand:
		return c.listEmails(ctx, v)
	default:
		return nil, fmt.Errorf("unsupported gmail command: %s", cmd.Name())
	}
}

type SendEmailCommand struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	From    string   `json:"from,omitempty"`
}

func (SendEmailCommand) Name() string { return "gmail.send_email" }
func (c SendEmailCommand) Args() map[string]any {
	return map[string]any{"to": c.To, "subject": c.Subject, "body": c.Body, "from": c.From}
}

type ListEmailsCommand struct {
	Query string `json:"query,omitempty"`
	Max   int    `json:"max,omitempty"`
}

func (ListEmailsCommand) Name() string { return "gmail.list_emails" }
func (c ListEmailsCommand) Args() map[string]any {
	return map[string]any{"query": c.Query, "max": c.Max}
}

func (c *GmailCapability) sendEmail(ctx context.Context, cmd SendEmailCommand) (*capability.Result, error) {
	if len(cmd.To) == 0 || cmd.Subject == "" || cmd.Body == "" {
		return nil, fmt.Errorf("to, subject and body are required")
	}
	raw := buildRFC822(cmd)
	message := map[string]any{"raw": base64.RawURLEncoding.EncodeToString([]byte(raw))}
	var sent map[string]any
	if err := c.doJSON(ctx, http.MethodPost, "/users/me/messages/send", message, &sent); err != nil {
		return nil, err
	}
	return capability.NewResult(cmd.Name(), map[string]any{"message": sent}), nil
}

func (c *GmailCapability) listEmails(ctx context.Context, cmd ListEmailsCommand) (*capability.Result, error) {
	path := "/users/me/messages?maxResults=20"
	if cmd.Query != "" {
		path += "&q=" + cmd.Query
	}
	if cmd.Max > 0 {
		path += fmt.Sprintf("&maxResults=%d", cmd.Max)
	}
	var response struct {
		Messages []map[string]any `json:"messages"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return capability.NewResult(cmd.Name(), map[string]any{"messages": response.Messages}), nil
}

func buildRFC822(cmd SendEmailCommand) string {
	var b strings.Builder
	if cmd.From != "" {
		b.WriteString("From: " + cmd.From + "\r\n")
	}
	b.WriteString("To: " + strings.Join(cmd.To, ", ") + "\r\n")
	b.WriteString("Subject: " + cmd.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
	b.WriteString(cmd.Body)
	return b.String()
}

func (c *GmailCapability) doJSON(ctx context.Context, method, path string, body any, out any) error {
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reader)
	if err != nil {
		return err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gmail request failed: %s: %s", resp.Status, string(data))
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}
