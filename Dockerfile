FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN go build -o bot ./cmd/bot

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Jakarta

WORKDIR /app

COPY --from=builder /app/bot .

RUN mkdir -p /app/data/whatsapp /app/data/google

# Menulis file JSON otomatis dari env GOOGLE_CREDENTIALS_JSON jika tersedia
CMD ["sh", "-c", "if [ -n \"$GOOGLE_CREDENTIALS_JSON\" ]; then echo \"$GOOGLE_CREDENTIALS_JSON\" > /app/data/google/service-account.json; fi && ./bot"]