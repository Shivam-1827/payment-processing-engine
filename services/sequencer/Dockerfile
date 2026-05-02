# Build Stage
FROM debian:bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    cmake g++ librdkafka-dev nlohmann-json3-dev

WORKDIR /app
COPY services/sequencer/ .

# Build the C++ Engine
RUN cmake -S . -B build && cmake --build build --config Release

# Runtime Stage (Slim image with only runtime libraries)
FROM debian:bookworm-slim

# Install the C++ runtime Kafka library (this automatically pulls in the C library too)
RUN apt-get update && apt-get install -y librdkafka++1 && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/build/sequencer .

CMD ["./sequencer"]