FROM golang:1.14.9-alpine AS builder
RUN mkdir /build
ADD api /build/
WORKDIR /build
RUN go build

FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/go-reddit /app/
COPY api/static /app/static
WORKDIR /app
CMD ["./go-reddit"]