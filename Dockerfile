FROM golang:1.22.4-alpine AS builder

WORKDIR /app

RUN go install std

COPY go.* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ./build/api-server ./cmd/api-server


FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/build/api-server .
COPY --from=builder /app/.env .

CMD ["./api-server", "-cfg", ".env"]