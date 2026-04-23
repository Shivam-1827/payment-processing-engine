package transport

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// --- Mocks ---
type MockIdempStore struct{}
func (m *MockIdempStore) AcquireLock(ctx context.Context, key string) (bool, string, error) {
	return true, "", nil // Always simulate a brand new request
}
func (m *MockIdempStore) SaveResponse(ctx context.Context, key string, res string) error { return nil }
func (m *MockIdempStore) Delete(ctx context.Context, key string) error { return nil }

type MockProducer struct{}
func (m *MockProducer) PublishIntent(ctx context.Context, key string, payload any) error { return nil }
func (m *MockProducer) Close() error { return nil }

// --- Tests ---
func TestHandleCreatePayment_Success(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handler := NewPaymentHandler(&MockIdempStore{}, &MockProducer{}, logger)

	// Valid Payload
	body := []byte(`{"from_account":"123e4567-e89b-12d3-a456-426614174000", "to_account":"123e4567-e89b-12d3-a456-426614174001", "amount":1000, "currency":"USD"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/payments", bytes.NewBuffer(body))
	req.Header.Set("Idempotency-Key", "test-key-123")

	rr := httptest.NewRecorder()
	handler.HandleCreatePayment(rr, req)

	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
	}
}

func TestHandleCreatePayment_InvalidAmount(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	handler := NewPaymentHandler(&MockIdempStore{}, &MockProducer{}, logger)

	// Invalid Payload (Negative Amount)
	body := []byte(`{"from_account":"123e4567-e89b-12d3-a456-426614174000", "to_account":"123e4567-e89b-12d3-a456-426614174001", "amount":-500, "currency":"USD"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/payments", bytes.NewBuffer(body))
	req.Header.Set("Idempotency-Key", "test-key-123")

	rr := httptest.NewRecorder()
	handler.HandleCreatePayment(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}