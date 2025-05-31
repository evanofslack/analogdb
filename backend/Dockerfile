FROM golang:1.24-alpine AS builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN go build cmd/analogdb/main.go
# RUN go build -o main ./cmd/analogdb

FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/main /app/
COPY static/ /app/static
COPY /config/config.yml  /app/
WORKDIR /app
CMD ["./main"]
