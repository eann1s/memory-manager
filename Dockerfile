FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o mm ./cmd/mm

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/mm /app/mm

ENV HTTP_PORT=:8080
EXPOSE 8080

CMD ["/app/mm"]
