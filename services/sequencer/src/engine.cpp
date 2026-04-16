#include "engine.hpp"
#include <iostream>
#include <ctime>

json SequencerEngine::ProcessIntent(const std::string& idempotency_key, const json& payload) {
    json output_event;
    output_event["idempotency_key"] = idempotency_key;
    output_event["timestamp"] = std::time(nullptr);
    output_event["payload"] = payload;

    // 1. Strict State Machine Check
    if (active_transactions.find(idempotency_key) != active_transactions.end()) {
        // We have already sequenced this! Duplicate event from Kafka retry.
        output_event["status"] = "REJECTED";
        output_event["reason"] = "DUPLICATE_SEQUENCE";
        return output_event;
    }

    // 2. Register State (In-Memory Microsecond Operation)
    active_transactions[idempotency_key] = TxnState::INIT;

    // 3. Fast-Path Validations (e.g., Velocity checks, basic fraud bounds)
    long amount = payload.value("amount", 0);
    if (amount > 100000000) { // Reject transactions over $1M instantly
        active_transactions[idempotency_key] = TxnState::REJECTED;
        output_event["status"] = "REJECTED";
        output_event["reason"] = "AMOUNT_EXCEEDS_LIMIT";
        return output_event;
    }

    // 4. State Transition
    active_transactions[idempotency_key] = TxnState::VALIDATED;
    
    // Output the validated event to be picked up by the Go Orchestrator
    output_event["status"] = "VALIDATED";
    return output_event;
}