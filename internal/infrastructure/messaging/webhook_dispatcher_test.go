package messaging

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/event"
)

// testEvent implements event.DomainEvent for testing.
type testEvent struct {
	eventType   string
	aggregateID string
	occurredAt  time.Time
	payload     map[string]interface{}
}

func (e *testEvent) Type() string                  { return e.eventType }
func (e *testEvent) AggregateID() string           { return e.aggregateID }
func (e *testEvent) OccurredAt() time.Time         { return e.occurredAt }
func (e *testEvent) Payload() map[string]interface{} { return e.payload }

func newTestEvent(eventType, aggregateID string) *testEvent {
	return &testEvent{
		eventType:   eventType,
		aggregateID: aggregateID,
		occurredAt:  time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC),
		payload:     map[string]interface{}{"key": "value"},
	}
}

func TestWebhookDispatcher_DelegatesToInner(t *testing.T) {
	inner := event.NewInMemoryDispatcher()
	d := NewWebhookDispatcher(inner, nil)

	evt := newTestEvent("instance.created", "agg-1")
	err := d.Dispatch(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inner.Count() != 1 {
		t.Fatalf("expected inner to have 1 event, got %d", inner.Count())
	}
}

func TestWebhookDispatcher_DispatchBatchDelegatesToInner(t *testing.T) {
	inner := event.NewInMemoryDispatcher()
	d := NewWebhookDispatcher(inner, nil)

	events := []event.DomainEvent{
		newTestEvent("instance.created", "agg-1"),
		newTestEvent("instance.completed", "agg-2"),
	}
	err := d.DispatchBatch(context.Background(), events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inner.Count() != 2 {
		t.Fatalf("expected inner to have 2 events, got %d", inner.Count())
	}
}

func TestWebhookDispatcher_InnerErrorTakesPrecedence(t *testing.T) {
	inner := &failingDispatcher{err: fmt.Errorf("inner error")}
	d := NewWebhookDispatcher(inner, nil)

	evt := newTestEvent("instance.created", "agg-1")
	err := d.Dispatch(context.Background(), evt)
	if err == nil || err.Error() != "inner error" {
		t.Fatalf("expected inner error, got: %v", err)
	}
}

func TestWebhookDispatcher_DeliversToMatchingWebhook(t *testing.T) {
	var mu sync.Mutex
	var received []webhookPayload

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p webhookPayload
		json.Unmarshal(body, &p)
		mu.Lock()
		received = append(received, p)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	inner := event.NewInMemoryDispatcher()
	webhooks := []WebhookConfig{
		{
			URL:    srv.URL,
			Events: []string{"instance.created"},
			Active: true,
		},
	}
	d := NewWebhookDispatcher(inner, webhooks)

	evt := newTestEvent("instance.created", "agg-1")
	err := d.Dispatch(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for async delivery
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(received))
	}
	if received[0].EventType != "instance.created" {
		t.Errorf("expected event_type instance.created, got %s", received[0].EventType)
	}
	if received[0].AggregateID != "agg-1" {
		t.Errorf("expected aggregate_id agg-1, got %s", received[0].AggregateID)
	}
}

func TestWebhookDispatcher_SkipsNonMatchingEvents(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	inner := event.NewInMemoryDispatcher()
	webhooks := []WebhookConfig{
		{
			URL:    srv.URL,
			Events: []string{"instance.completed"},
			Active: true,
		},
	}
	d := NewWebhookDispatcher(inner, webhooks)

	evt := newTestEvent("instance.created", "agg-1")
	_ = d.Dispatch(context.Background(), evt)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 0 {
		t.Fatalf("expected 0 deliveries for non-matching event, got %d", callCount)
	}
}

func TestWebhookDispatcher_SkipsInactiveWebhooks(t *testing.T) {
	callCount := 0
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	inner := event.NewInMemoryDispatcher()
	webhooks := []WebhookConfig{
		{
			URL:    srv.URL,
			Events: []string{"instance.created"},
			Active: false,
		},
	}
	d := NewWebhookDispatcher(inner, webhooks)

	evt := newTestEvent("instance.created", "agg-1")
	_ = d.Dispatch(context.Background(), evt)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 0 {
		t.Fatalf("expected 0 deliveries for inactive webhook, got %d", callCount)
	}
}

func TestWebhookDispatcher_HMACSignature(t *testing.T) {
	secret := "my-secret-key"
	var capturedSignature string
	var capturedBody []byte
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		capturedSignature = r.Header.Get("X-FlowEngine-Signature")
		capturedBody, _ = io.ReadAll(r.Body)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	inner := event.NewInMemoryDispatcher()
	webhooks := []WebhookConfig{
		{
			URL:    srv.URL,
			Events: []string{"instance.created"},
			Secret: secret,
			Active: true,
		},
	}
	d := NewWebhookDispatcher(inner, webhooks)

	evt := newTestEvent("instance.created", "agg-1")
	_ = d.Dispatch(context.Background(), evt)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Verify HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(capturedBody)
	expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if capturedSignature != expectedSig {
		t.Errorf("expected signature %s, got %s", expectedSig, capturedSignature)
	}
}

func TestWebhookDispatcher_CustomHeaders(t *testing.T) {
	var capturedHeaders http.Header
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		capturedHeaders = r.Header.Clone()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	inner := event.NewInMemoryDispatcher()
	webhooks := []WebhookConfig{
		{
			URL:     srv.URL,
			Events:  []string{"instance.created"},
			Headers: map[string]string{"X-Custom": "test-value"},
			Active:  true,
		},
	}
	d := NewWebhookDispatcher(inner, webhooks)

	evt := newTestEvent("instance.created", "agg-1")
	_ = d.Dispatch(context.Background(), evt)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if capturedHeaders.Get("X-Custom") != "test-value" {
		t.Errorf("expected X-Custom header 'test-value', got '%s'", capturedHeaders.Get("X-Custom"))
	}
	if capturedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got '%s'", capturedHeaders.Get("Content-Type"))
	}
}

func TestWebhookDispatcher_RetriesOnFailure(t *testing.T) {
	var mu sync.Mutex
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		count := callCount
		mu.Unlock()
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	inner := event.NewInMemoryDispatcher()
	webhooks := []WebhookConfig{
		{
			URL:        srv.URL,
			Events:     []string{"instance.created"},
			MaxRetries: 3,
			Active:     true,
		},
	}
	d := NewWebhookDispatcher(inner, webhooks)

	evt := newTestEvent("instance.created", "agg-1")
	_ = d.Dispatch(context.Background(), evt)

	// Wait for retries (1s + 2s backoff + some buffer)
	time.Sleep(5 * time.Second)

	mu.Lock()
	defer mu.Unlock()
	if callCount < 3 {
		t.Errorf("expected at least 3 attempts, got %d", callCount)
	}
}

func TestWebhookDispatcher_EmptyEventsMatchesAll(t *testing.T) {
	var mu sync.Mutex
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	inner := event.NewInMemoryDispatcher()
	webhooks := []WebhookConfig{
		{
			URL:    srv.URL,
			Events: []string{}, // empty = match all
			Active: true,
		},
	}
	d := NewWebhookDispatcher(inner, webhooks)

	evt := newTestEvent("instance.created", "agg-1")
	_ = d.Dispatch(context.Background(), evt)

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if callCount != 1 {
		t.Fatalf("expected 1 delivery for wildcard webhook, got %d", callCount)
	}
}

// failingDispatcher always returns an error.
type failingDispatcher struct {
	err error
}

func (d *failingDispatcher) Dispatch(_ context.Context, _ event.DomainEvent) error {
	return d.err
}

func (d *failingDispatcher) DispatchBatch(_ context.Context, _ []event.DomainEvent) error {
	return d.err
}
