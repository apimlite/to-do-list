FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o myapp .

FROM alpine:3.20.2

# Create a non-root user and group
# Using a fixed UID/GID for better security and predictability
RUN addgroup -g 10014 appgroup && \
    adduser -u 10014 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/myapp /app/aws-marketplace-integration
COPY --from=builder /app/resources/static /app/resources/static

# Change ownership of the application files to the non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER 10014

EXPOSE 8080

ENTRYPOINT ["/app/aws-marketplace-integration"]
