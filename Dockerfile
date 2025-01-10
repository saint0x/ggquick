# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy only the necessary files for building
COPY go.mod go.sum ./
COPY pkg ./pkg/
COPY cmd ./cmd/
COPY sysprompt.json ./

# Build the application
RUN go build -o server ./cmd/server

# Runtime stage
FROM alpine:latest AS final

WORKDIR /app

# Copy the binary and config from builder
COPY --from=builder /app/server .
COPY --from=builder /app/sysprompt.json .

# Create empty config file
RUN echo '{"repo_url":"","owner":"","name":""}' > ggquick.json

# Expose the port
EXPOSE 8080

# Run the server
CMD ["./server"] 