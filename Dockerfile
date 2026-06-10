# Multi-stage build for taskdock (Go + React).
# Stage 1: Build the React frontend with Vite.
# Stage 2: Build a fully static Go binary with CGO (required for SQLite).
# Stage 3: scratch image — no OS, no shell, just the binary + static assets.

# --- Frontend build stage ---
FROM node:22-alpine AS frontend

WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# --- Go build stage ---
FROM golang:1.26-alpine AS builder

# gcc + musl-dev are required because mattn/go-sqlite3 is a CGO package.
RUN apk add --no-cache gcc musl-dev

WORKDIR /src

# Cache dependencies — this layer only rebuilds when go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Build a fully static binary so it runs in scratch without libc.
COPY . .
RUN CGO_ENABLED=1 go build \
    -ldflags="-s -w -linkmode external -extldflags '-static'" \
    -o /taskdock ./cmd/server/main.go \
    && mkdir /empty

# --- Runtime stage ---
FROM scratch

WORKDIR /app

COPY --from=builder /taskdock /app/taskdock
COPY --from=frontend /src/dist /app/static

# scratch has no mkdir — create /data by copying an empty dir from builder.
# docker-compose mounts a volume here at runtime.
COPY --from=builder /empty /data

EXPOSE 8860
ENV DATA_DIR=/data

ENTRYPOINT ["/app/taskdock"]
