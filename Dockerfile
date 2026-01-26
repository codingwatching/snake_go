# Build Stage
FROM golang:1.23-bookworm AS builder

# Install build dependencies
RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources && \
    apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

# Copy source code
COPY . .

# Build the application
# We use CGO_ENABLED=1 because onnxruntime_go requires it for dynamic library interaction
RUN CGO_ENABLED=1 GOOS=linux go build -v -o webserver ./cmd/webserver/main.go

# Runtime Stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources && \
    apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Download and install ONNX Runtime shared library
# Version 1.19.2 is a stable recent version
ENV ORT_VERSION=1.19.2
RUN curl -L https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-linux-x64-${ORT_VERSION}.tgz -o ort.tgz \
    && tar -xzf ort.tgz \
    && cp onnxruntime-linux-x64-${ORT_VERSION}/lib/libonnxruntime.so.${ORT_VERSION} /usr/lib/libonnxruntime.so \
    && rm -rf onnxruntime-linux-x64-${ORT_VERSION} ort.tgz

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/webserver .

# Copy static files and models
COPY --from=builder /app/web/static ./web/static
COPY --from=builder /app/ml/checkpoints ./ml/checkpoints

# Expose the game server port
EXPOSE 8080

# Environment variable for detailed logs (optional, can be overridden)
ENV DETAILED_LOGS=true

# Start the server
# We use a shell form to allow environment variable expansion if needed, 
# but EXEC form is preferred for signal handling.
CMD ["./webserver", "--detailed-logs"]
