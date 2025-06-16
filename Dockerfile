FROM golang:1.23-alpine as builder

ENV TZ=Europe/Moscow

COPY . /application

WORKDIR /application

RUN go mod download
RUN go build -o ./build ./cmd/main.go

FROM alpine:latest

ENV TZ=Europe/Moscow

WORKDIR /application

COPY --from=builder /application/build ./build
COPY --from=builder /application/env ./env

EXPOSE 80

CMD ["/application/build"]