FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
RUN mkdir /build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main cmd/analogdb-consumer/main.go

FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/main /app/
COPY internal/config/config.yaml /app/
WORKDIR /app
CMD ["./main"]
