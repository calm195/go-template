FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /server ./server
COPY configs/config.yaml ./configs/config.yaml
USER app
EXPOSE 8080
ENTRYPOINT ["./server"]
