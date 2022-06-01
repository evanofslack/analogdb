FROM golang:1.18.2-alpine as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN cd cmd/analogdb && go build

FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/analogdb /app/
COPY static/ /app/static
COPY .env  /app/
WORKDIR /app
CMD ["./analogdb"]