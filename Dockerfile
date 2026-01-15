# --- Stage 1: Build (Same as before) ---
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
# We MUST build a static binary (CGO_ENABLED=0) for distroless
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o celerix-stored ./cmd/celerix-stored/main.go

# --- Stage 2: Distroless Final Image ---
# "static" is the smallest version, containing only tzdata and ca-certs
FROM gcr.io/distroless/static-debian12

WORKDIR /

# Copy the binary from the builder
COPY --from=builder /app/celerix-stored /celerix-stored

# Distroless doesn't have a shell, so we must use the "exec" form
# We also ensure the data directory is handled via a Volume in Compose
EXPOSE 7000

ENTRYPOINT ["/celerix-stored"]