#pragma once
#include <string>
#include <unordered_map>
#include <nlohmann/json.hpp>

using json = nlohmann::json;

enum class TxnState {
    INIT,
    VALIDATED,
    REJECTED
};

class SequencerEngine {
private:
    // In a real HFT system, this would be a highly optimized ring buffer or lock-free map.
    // We store the active state of transactions in-memory to bypass database lookups.
    std::unordered_map<std::string, TxnState> active_transactions;

public:
    SequencerEngine() = default;

    // Process evaluates the intent, transitions the state, and returns the output event
    json ProcessIntent(const std::string& idempotency_key, const json& payload);
};