FROM golang:1.24.2-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# ---------- Runtime Stage ----------
FROM alpine:latest

# Install certs
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Switch to the new user
USER appuser

# Use appuser's home as working dir
WORKDIR /home/appuser/

# Copy binary from builder stage
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]