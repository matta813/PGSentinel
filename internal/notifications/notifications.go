package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Message struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	Severity string `json:"severity"`
	URL      string `json:"url,omitempty"`
}
type Provider interface {
	Send(context.Context, Message) error
}
type Webhook struct {
	URL     string
	Headers map[string]string
	client  *http.Client
}

func NewWebhook(target string, headers map[string]string) (*Webhook, error) {
	if err := safeURL(target); err != nil {
		return nil, err
	}
	return &Webhook{URL: target, Headers: headers, client: &http.Client{Timeout: 10 * time.Second}}, nil
}
func (w *Webhook) Send(ctx context.Context, m Message) error {
	body, _ := json.Marshal(m)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.Headers {
		req.Header.Set(k, v)
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}

type Ntfy struct {
	ServerURL, Topic, Token, Username, Password string
	client                                      *http.Client
}

func NewNtfy(serverURL, topic string) *Ntfy {
	return &Ntfy{ServerURL: strings.TrimRight(serverURL, "/"), Topic: topic, client: &http.Client{Timeout: 10 * time.Second}}
}
func (n *Ntfy) Send(ctx context.Context, m Message) error {
	if err := safeURL(n.ServerURL); err != nil {
		return err
	}
	payload := map[string]any{"topic": n.Topic, "title": m.Title, "message": m.Body, "priority": priority(m.Severity), "tags": []string{"elephant", "warning"}}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.ServerURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if n.Token != "" {
		req.Header.Set("Authorization", "Bearer "+n.Token)
	} else if n.Username != "" {
		req.SetBasicAuth(n.Username, n.Password)
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned HTTP %d", resp.StatusCode)
	}
	return nil
}
func priority(s string) int {
	switch s {
	case "CRITICAL":
		return 5
	case "HIGH":
		return 4
	case "MEDIUM":
		return 3
	default:
		return 2
	}
}
func safeURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return fmt.Errorf("provider URL must be an absolute HTTP(S) URL")
	}
	return nil
}
