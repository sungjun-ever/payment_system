FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY  . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -v -o /app/bin/worker ./cmd/worker

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/bin/api /usr/local/bin/api
COPY --from=builder /app/bin/worker /usr/local/bin/worker

EXPOSE 8080

CMD ["/usr/local/bin/api"]
