cmake_minimum_required(VERSION 3.20)
project(PaymentSequencer VERSION 1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -03 -Wall -Wextra -pthread")

find_package(RDKafka REQUIRED)
find_package(nlohmann_json REQUIRED)

add_executable(sequencer
    src/main.cpp
    src/engine.cpp
    src/kafka_client.cpp
)

target_link_libraries(sequencer PRIVATE RdKafka::rdkafka++ nlohmann_json::nlohmann_json)