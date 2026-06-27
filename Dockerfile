FROM node:24-slim AS ui
WORKDIR /app/ui
RUN corepack enable
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY ui/ .
RUN pnpm build

FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Install swag as a binary so it's cached and available without "go run" dep resolution
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6
RUN swag init -g cmd/server/main.go -o docs
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=ui /app/ui/dist ./ui/dist
EXPOSE 8080
ENTRYPOINT ["./server"]