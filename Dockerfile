FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
    -X main.Version=${VERSION} \
    -X main.GitCommit=${GIT_COMMIT} \
    -X main.BuildDate=${BUILD_DATE}" \
    -o /nothingllm ./cmd/nothingllm

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /nothingllm /nothingllm
COPY --from=builder /app/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --retries=5 CMD wget -qO- http://localhost:8080/health || exit 1
ENTRYPOINT ["/entrypoint.sh"]