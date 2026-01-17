# Stage 1: Build the frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build the backend
FROM golang:1.25.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy the built frontend to the backend directory for embedding
COPY --from=frontend-builder /app/frontend/dist ./cmd/celerix-stored/dist
# Run tests during build
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o celerix-stored ./cmd/celerix-stored/main.go

# Stage 3: Final Image
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/celerix-stored .
RUN mkdir -p data
EXPOSE 7001 7002
ENV CELERIX_PORT=7001
ENV CELERIX_HTTP_PORT=7002
ENV CELERIX_DATA_DIR=/app/data
CMD ["./celerix-stored"]