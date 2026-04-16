#include "kafka_client.hpp"
#include <iostream>
#include <stdexcept>

void DeliveryReportCb::dr_cb(RdKafka::Message &message) {
    if (message.err() != RdKafka::ERR_NO_ERROR) {
        std::cerr << "[Kafka Error] Message delivery failed: " << message.errstr() << std::endl;
    }
}

KafkaClient::KafkaClient(const std::string& brokers, const std::string& group_id, 
                         const std::string& in_topic, const std::string& out_topic) 
    : out_topic_(out_topic) {
    
    std::string errstr;
    std::unique_ptr<RdKafka::Conf> conf(RdKafka::Conf::create(RdKafka::Conf::CONF_GLOBAL));

    conf->set("metadata.broker.list", brokers, errstr);
    conf->set("group.id", group_id, errstr);
    conf->set("auto.offset.reset", "earliest", errstr);
    
    // CRITICAL: Disable auto-commit. We only commit after successfully producing the output event.
    conf->set("enable.auto.commit", "false", errstr); 
    conf->set("dr_cb", &dr_cb_, errstr);

    // Initialize Consumer
    consumer_.reset(RdKafka::KafkaConsumer::create(conf.get(), errstr));
    if (!consumer_) {
        throw std::runtime_error("Failed to create consumer: " + errstr);
    }

    // Initialize Producer
    producer_.reset(RdKafka::Producer::create(conf.get(), errstr));
    if (!producer_) {
        throw std::runtime_error("Failed to create producer: " + errstr);
    }

    // Subscribe to input topic
    std::vector<std::string> topics = {in_topic};
    RdKafka::ErrorCode err = consumer_->subscribe(topics);
    if (err != RdKafka::ERR_NO_ERROR) {
        throw std::runtime_error("Failed to subscribe to topic: " + RdKafka::err2str(err));
    }
}

KafkaClient::~KafkaClient() {
    consumer_->close();
    // Flush the producer queue before shutting down to prevent message loss
    producer_->flush(5000); 
}

RdKafka::Message* KafkaClient::Consume(int timeout_ms) {
    return consumer_->consume(timeout_ms);
}

bool KafkaClient::Produce(const std::string& key, const std::string& payload) {
    RdKafka::ErrorCode resp = producer_->produce(
        out_topic_, RdKafka::Topic::PARTITION_UA,
        RdKafka::Producer::RK_MSG_COPY,
        const_cast<char *>(payload.c_str()), payload.size(),
        key.c_str(), key.size(),
        0, nullptr, nullptr);

    if (resp != RdKafka::ERR_NO_ERROR) {
        std::cerr << "[Kafka Error] Produce failed: " << RdKafka::err2str(resp) << std::endl;
        return false;
    }

    // CRITICAL: Poll the producer to trigger delivery report callbacks. 
    // Without this, the outbound queue fills up and the application stalls.
    producer_->poll(0); 
    return true;
}

void KafkaClient::Commit(RdKafka::Message* msg) {
    RdKafka::ErrorCode err = consumer_->commitSync(msg);
    if (err != RdKafka::ERR_NO_ERROR) {
        std::cerr << "[Kafka Error] Failed to commit offset: " << RdKafka::err2str(err) << std::endl;
    }
}