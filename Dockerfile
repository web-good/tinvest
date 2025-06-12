FROM golang1.24-alpine3.22 as builder

LABEL authors="oleg"

ENTRYPOINT ["top", "-b"]