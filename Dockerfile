FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o /teltonika-tracker main.go
RUN go build -o /teltonika-server ./cmd/server

FROM alpine:3.18
RUN apk add --no-cache ca-certificates

COPY --from=builder /teltonika-tracker /teltonika-tracker
COPY --from=builder /teltonika-server /teltonika-server
WORKDIR /app

CMD ["/teltonika-tracker"]
