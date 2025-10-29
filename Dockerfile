# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o taskflow main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# Create non-root user (security best practice)
RUN adduser -D -s /bin/sh appuser
USER appuser

COPY --from=builder /app/taskflow .


EXPOSE 8080
CMD ["./taskflow"]