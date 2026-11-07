FROM golang:1.27.3-alpine

# Tools: git (untuk go modules tertentu), air (hot reload)
RUN apk add --no-cache git \
    && go install github.com/air-verse/air@v1.67.4

WORKDIR /app

# Cache dependency; source code di-bind-mount lewat compose.dev.yaml
COPY go.mod go.sum ./
RUN go mod download

EXPOSE 3000

CMD ["air", "-c", ".air.toml"]
