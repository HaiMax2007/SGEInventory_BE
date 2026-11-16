# ---- Build stage ----
FROM golang:1.27.3-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/app

# ---- Runtime stage ----
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/app /app/app

# Switch to non-root user
USER app

EXPOSE 3000

ENTRYPOINT ["/app/app"]
