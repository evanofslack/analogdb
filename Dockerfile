# FROM golang:1.14.9-alpine AS builder
FROM golang:1.18.2-alpine as builder
RUN mkdir /build
ADD server /build/
WORKDIR /build
RUN go build

FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/go-reddit /app/
COPY server/static/ /app/static
COPY server/.env  /app/
WORKDIR /app
CMD ["./go-reddit"]