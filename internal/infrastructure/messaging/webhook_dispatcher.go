package messaging

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/logger"
)

// WebhookConfig represents a webhook subscription.
type WebhookConfig struct {
	URL        string
	Events     []string          // Event types to subscribe to
	Secret     string            // HMAC secret for signing payloads
	Headers    map[string]string // Custom headers
	MaxRetries int
	Active     bool
}

// webhookPayload is the JSON structure sent to webhook endpoints.
type webhookPayload struct {
	EventType   string                 `json:"event_type"`
	AggregateID string                 `json:"aggregate_id"`
	OccurredAt  time.Time              `json:"occurred_at"`
	Payload     map[string]interface{} `json:"payload"`
}

// WebhookDispatcher wraps an inner dispatcher and sends webhooks for matching events.
type WebhookDispatcher struct {
	inner    event.Dispatcher
	webhooks []WebhookConfig
	client   *http.Client
}

// NewWebhookDispatcher creates a new WebhookDispatcher that delegates to the inner
// dispatcher and asynchronously delivers webhooks for matching events.
func NewWebhookDispatcher(inner event.Dispatcher, webhooks []WebhookConfig) *WebhookDispatcher {
	return &WebhookDispatcher{
		inner:    inner,
		webhooks: webhooks,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Dispatch delegates to the inner dispatcher, then asynchronously delivers webhooks.
func (d *WebhookDispatcher) Dispatch(ctx context.Context, evt event.DomainEvent) error {
	if err := d.inner.Dispatch(ctx, evt); err != nil {
		return err
	}
	d.deliverWebhooksAsync(evt)
	return nil
}

// DispatchBatch delegates to the inner dispatcher, then asynchronously delivers webhooks
// for each event in the batch.
func (d *WebhookDispatcher) DispatchBatch(ctx context.Context, events []event.DomainEvent) error {
	if err := d.inner.DispatchBatch(ctx, events); err != nil {
		return err
	}
	for _, evt := range events {
		d.deliverWebhooksAsync(evt)
	}
	return nil
}

// deliverWebhooksAsync fires off goroutines to deliver webhooks for the given event.
func (d *WebhookDispatcher) deliverWebhooksAsync(evt event.DomainEvent) {
	for _, wh := range d.webhooks {
		if !wh.Active {
			continue
		}
		if !d.eventMatches(wh, evt) {
			continue
		}
		go d.deliverWebhook(wh, evt)
	}
}

// eventMatches returns true if the webhook is subscribed to the given event type.
// An empty Events list means the webhook subscribes to all events.
func (d *WebhookDispatcher) eventMatches(wh WebhookConfig, evt event.DomainEvent) bool {
	if len(wh.Events) == 0 {
		return true
	}
	for _, t := range wh.Events {
		if t == evt.Type() {
			return true
		}
	}
	return false
}

// deliverWebhook sends the event to the webhook URL with retries and HMAC signing.
func (d *WebhookDispatcher) deliverWebhook(wh WebhookConfig, evt event.DomainEvent) {
	payload := webhookPayload{
		EventType:   evt.Type(),
		AggregateID: evt.AggregateID(),
		OccurredAt:  evt.OccurredAt(),
		Payload:     evt.Payload(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Warn("webhook: failed to marshal payload",
			"url", wh.URL,
			"event_type", evt.Type(),
			"error", err,
		)
		return
	}

	maxAttempts := wh.MaxRetries + 1
	backoff := 1 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = d.sendRequest(wh, body)
		if err == nil {
			logger.Debug("webhook: delivered successfully",
				"url", wh.URL,
				"event_type", evt.Type(),
				"attempt", attempt,
			)
			return
		}

		logger.Warn("webhook: delivery failed",
			"url", wh.URL,
			"event_type", evt.Type(),
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"error", err,
		)

		if attempt < maxAttempts {
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	logger.Warn("webhook: all delivery attempts exhausted",
		"url", wh.URL,
		"event_type", evt.Type(),
		"max_retries", wh.MaxRetries,
	)
}

// sendRequest performs a single HTTP POST to the webhook URL.
func (d *WebhookDispatcher) sendRequest(wh WebhookConfig, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Apply custom headers
	for k, v := range wh.Headers {
		req.Header.Set(k, v)
	}

	// HMAC signing
	if wh.Secret != "" {
		mac := hmac.New(sha256.New, []byte(wh.Secret))
		mac.Write(body)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-FlowEngine-Signature", "sha256="+signature)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}
