FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
# Install pnpm
RUN npm install -g pnpm

# Copy frontend files
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install

COPY frontend/ ./
RUN pnpm run build

# Go builder stage
FROM golang:1.22-alpine AS backend-builder

WORKDIR /app
COPY backend/ ./backend/
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

WORKDIR /app/backend
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/build/system-monitor

# Final stage
FROM alpine:latest

WORKDIR /app
COPY --from=backend-builder /app/build/system-monitor .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Install necessary system utilities for metrics collection
RUN apk add --no-cache procps iproute2 sysstat

EXPOSE 3000

CMD ["./system-monitor"] 