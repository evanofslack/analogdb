FROM golang:1.20-alpine as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN go build cmd/analogdb/main.go

FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/main /app/
COPY static/ /app/static
COPY /config/config.yml  /app/
WORKDIR /app
CMD ["./main"]
