FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/lowkeymycity ./cmd/lowkeymycity
RUN GOBIN=/out go install github.com/pressly/goose/v3/cmd/goose@latest

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=builder /out/lowkeymycity /app/lowkeymycity
COPY --from=builder /out/goose /usr/local/bin/goose
COPY migrations/ /app/migrations/
USER app
CMD ["/app/lowkeymycity"]
