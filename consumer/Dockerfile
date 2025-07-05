FROM golang:1.24-alpine AS builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN go build cmd/analogdb-consumer/main.go

FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/main /app/
COPY /config/config.yml  /app/
WORKDIR /app
CMD ["./main"]
