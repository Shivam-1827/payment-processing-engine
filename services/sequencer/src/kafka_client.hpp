#pragma once

#include <librdkafka/rdkafkacpp.h>
#include <string>
#include <memory>

// Callback to track if our producer successfully delivered the message
class DeliveryReportCb : public RdKafka::DeliveryReportCb {
public:
    void dr_cb(RdKafka::Message &message) override;
};

class KafkaClient {
private:
    std::unique_ptr<RdKafka::KafkaConsumer> consumer_;
    std::unique_ptr<RdKafka::Producer> producer_;
    DeliveryReportCb dr_cb_;
    std::string out_topic_;

public:
    KafkaClient(const std::string& brokers, const std::string& group_id, 
                const std::string& in_topic, const std::string& out_topic);
    ~KafkaClient();

    // Fetches a message from the input topic
    RdKafka::Message* Consume(int timeout_ms);

    // Publishes a message to the output topic
    bool Produce(const std::string& key, const std::string& payload);

    // Synchronously commits the offset to guarantee exactly-once processing
    void Commit(RdKafka::Message* msg);
};