package transport

import (
"context"
"encoding/json"
"errors"
"log/slog"
"net/http"
"strings"

"github.com/google/uuid"
"github.com/Shivam-1827/payment-engine/services/api-gateway/internal/idempotency"
)

type PaymentHandler struct {
idempStore idempotency.Store
logger     *slog.Logger
}

func NewPaymentHandler(store idempotency.Store, logger *slog.Logger) *PaymentHandler {
return &PaymentHandler{idempStore: store, logger: logger}
}

// PaymentRequest with strict JSON tags
type PaymentRequest struct {
FromAccountID string `json:"from_account"`
ToAccountID   string `json:"to_account"`
Amount        int64  `json:"amount"`
Currency      string `json:"currency"`
}

// Validate sanitizes the input BEFORE we touch Redis or Kafka
func (req *PaymentRequest) Validate() error {
if req.Amount <= 0 {
return errors.New("amount must be greater than zero")
}
if len(req.Currency) != 3 {
return errors.New("currency must be a valid 3-letter ISO code")
}
req.Currency = strings.ToUpper(req.Currency)

if _, err := uuid.Parse(req.FromAccountID); err != nil {
return errors.New("from_account must be a valid UUID")
}
if _, err := uuid.Parse(req.ToAccountID); err != nil {
return errors.New("to_account must be a valid UUID")
}
if req.FromAccountID == req.ToAccountID {
return errors.New("from_account and to_account cannot be identical")
}
return nil
}

type PaymentResponse struct {
Status         string `json:"status"`
IdempotencyKey string `json:"idempotency_key"`
Message        string `json:"message"`
}

func (h *PaymentHandler) HandleCreatePayment(w http.ResponseWriter, r *http.Request) {
ctx := r.Context()
idempKey := r.Header.Get("Idempotency-Key")

// 1. Parse JSON First
var req PaymentRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
RespondError(w, http.StatusBadRequest, "Invalid JSON body structure")
return
}

// 2. Strict Input Validation Second
if err := req.Validate(); err != nil {
RespondError(w, http.StatusBadRequest, err.Error())
return
}

// 3. Acquire Distributed Lock THIRD
// Now we know the data is perfectly valid, it is safe to lock the key.
isNew, cachedRes, err := h.idempStore.AcquireLock(ctx, idempKey)
if err != nil {
if err == idempotency.ErrConcurrentRequest {
RespondError(w, http.StatusConflict, "Request is currently processing")
return
}
h.logger.Error("idempotency lock failure", "error", err, "key", idempKey)
RespondError(w, http.StatusInternalServerError, "Internal Server Error")
return
}

// 4. Return cached JSON if it's a valid retry
if !isNew {
h.logger.Info("idempotency cache hit", "key", idempKey)
w.Header().Set("Content-Type", "application/json")
w.WriteHeader(http.StatusOK)
w.Write([]byte(cachedRes))
return
}

// ---------------------------------------------------------
// ⚡ PHASE 3 PREVIEW: Publish to Kafka here
// This will never execute unless the payload is mathematically sound.
// err = h.kafkaProducer.Publish(ctx, "payment_intents", req)
// ---------------------------------------------------------

// 5. Construct Response and cache it
response := PaymentResponse{
Status:         idempotency.StatusProcessing,
IdempotencyKey: idempKey,
Message:        "Payment intent accepted and queued",
}

resBytes, _ := json.Marshal(response)

// Fire and forget the cache save (don't fail the request if saving to cache fails)
go func() {
_ = h.idempStore.SaveResponse(context.Background(), idempKey, string(resBytes))
}()

h.logger.Info("payment intent accepted", "idempotency_key", idempKey, "amount", req.Amount)
RespondJSON(w, http.StatusAccepted, response)
}
