FROM golang:1.25-alpine AS builder

ENV TZ=Europe/Moscow
COPY . /application

WORKDIR /application

RUN go mod download
ENV APP_ENV=prod
RUN go build -o ./build ./cmd/main.go

FROM alpine:latest

ENV TZ=Europe/Moscow

RUN apk add tzdata
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