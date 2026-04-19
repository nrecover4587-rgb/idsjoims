# REPLACE existing file at: Dockerfile
# (rename this file from Dockerfile.py to Dockerfile)

FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata curl

WORKDIR /app

RUN curl -o tg_public_keys.pem https://raw.githubusercontent.com/xelaj/mtproto/main/telegram/example_static/tg_public_keys.pem

COPY go.mod ./
COPY . .
RUN go mod tidy && go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o joinids-bot \
    ./cmd/main.go

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/joinids-bot .
COPY --from=builder /app/tg_public_keys.pem .

CMD ["./joinids-bot"]
