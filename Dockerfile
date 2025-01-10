# Build stage
FROM golang:1.21-alpine AS builder

# Add build argument to force rebuild
ARG CACHEBUST=1

# Set working directory
WORKDIR /app

# Copy only the necessary files for building
COPY go.mod go.sum ./
COPY pkg ./pkg/
COPY cmd ./cmd/

# Build the application
RUN go build -o server ./cmd/server

# Runtime stage
FROM alpine:latest AS final

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/server .

# Expose the port
EXPOSE 8080

# Run the server
CMD ["./server"] 