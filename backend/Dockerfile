FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/zxmail-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/zxmail-worker ./cmd/worker

FROM alpine:3.22

RUN apk add --no-cache wget
RUN adduser -D -u 10001 appuser
USER appuser
WORKDIR /app

COPY --from=builder /bin/zxmail-api /usr/local/bin/zxmail-api
COPY --from=builder /bin/zxmail-worker /usr/local/bin/zxmail-worker

EXPOSE 8080

CMD ["zxmail-api"]
