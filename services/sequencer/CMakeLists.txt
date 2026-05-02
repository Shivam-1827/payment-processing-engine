cmake_minimum_required(VERSION 3.20)
project(PaymentSequencer VERSION 1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -O3 -Wall -Wextra -pthread")

# nlohmann_json provides CMake configs via apt, so this stays
find_package(nlohmann_json REQUIRED)

add_executable(sequencer
    src/main.cpp
    src/engine.cpp
    src/kafka_client.cpp
)

# We link rdkafka++ directly using standard system paths provided by Debian apt
target_link_libraries(sequencer PRIVATE rdkafka++ nlohmann_json::nlohmann_json)