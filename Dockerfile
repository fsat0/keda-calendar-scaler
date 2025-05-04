FROM golang:1.24.2 AS builder

WORKDIR /src

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o calendar-scaler main.go


FROM alpine:latest

WORKDIR /

RUN apk add --no-cache tzdata

COPY --from=builder /src/calendar-scaler .

ENTRYPOINT ["/calendar-scaler"]
