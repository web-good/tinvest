FROM golang:1.25-alpine AS builder

ENV TZ=Europe/Moscow
COPY . /application

WORKDIR /application

RUN go mod download
ENV APP_ENV=prod
RUN go build -o ./build ./cmd/main.go

FROM alpine:latest

ENV TZ=Europe/Moscow

RUN apk add --no-cache tzdata ca-certificates

# The T-Bank Invest API certificate chains up to the Russian Ministry of Digital
# Development root CA, which ships in no upstream trust store — without this the
# gRPC handshake in pkg/client/grpc fails with "certificate signed by unknown
# authority" and every strategy loses market data. The certificate is vendored
# rather than downloaded at build time so the image does not depend on gu-st.ru
# being reachable, or on whatever it happens to serve.
# sha256: D2:6D:2D:02:31:B7:C3:9F:92:CC:73:85:12:BA:54:10:35:19:E4:40:5D:68:B5:BD:70:3E:97:88:CA:8E:CF:31
# Expires 2032-02-27. Only the root is needed; the API serves the intermediate.
COPY deploy/certs/russian_trusted_root_ca.crt /usr/local/share/ca-certificates/
RUN update-ca-certificates

ENV APP_ENV=prod
WORKDIR /application

COPY --from=builder /application/build ./build
COPY --from=builder /application/env ./env

# Reversion live runner persists per-ticker entry state here
# (data/state/reversion_<account>.json). Mount a persistent volume so the
# state survives container restarts/redeploys; otherwise it is lost and the
# runner falls back to approximate reconstruct-from-API on the next tick.
VOLUME ["/application/data/state"]

EXPOSE 50051

CMD ["/application/build"]