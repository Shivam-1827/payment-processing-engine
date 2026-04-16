#include <iostream>
#include <csignal>
#include <nlohmann/json.hpp>
#include "engine.hpp"
#include "kafka_client.hpp"

using json = nlohmann::json;
static bool run = true;

static void sigterm(int sig) {
    run = false;
}

int main() {
    // 1. Setup graceful shutdown signals
    std::signal(SIGINT, sigterm);
    std::signal(SIGTERM, sigterm);

    try {
        // 2. Initialize the Kafka Wrapper (Handles polling, callbacks, and memory safety)
        KafkaClient kafka("localhost:9092", "cpp_sequencer_group", "payment_intents", "payment_validated");
        SequencerEngine engine;

        std::cout << "🚀 C++ Sequencer running. Waiting for events..." << std::endl;

        // 3. The high-speed event loop
        while (run) {
            RdKafka::Message* msg = kafka.Consume(100); // 100ms timeout

            if (msg->err() == RdKafka::ERR_NO_ERROR) {
                std::string payload_str(static_cast<const char *>(msg->payload()), msg->len());
                std::string key_str = msg->key() ? *msg->key() : "";

                try {
                    // Step A: Parse and process business logic in memory
                    json incoming = json::parse(payload_str);
                    json outgoing = engine.ProcessIntent(key_str, incoming);

                    // Step B: Publish the validated result
                    if (kafka.Produce(key_str, outgoing.dump())) {
                        // Step C: Commit exactly-once ONLY if publish succeeded
                        kafka.Commit(msg);
                    }
                } catch (const std::exception& e) {
                    std::cerr << "[App Error] Processing failed: " << e.what() << std::endl;
                }
            }
            delete msg; // Crucial: Prevent memory leak from consumed messages
        }
    } catch (const std::exception& e) {
        std::cerr << "[Fatal Error] " << e.what() << std::endl;
        return 1;
    }

    std::cout << "Sequencer shutdown complete." << std::endl;
    // Wait for librdkafka background threads to exit cleanly
    RdKafka::wait_destroyed(5000); 
    return 0;
}