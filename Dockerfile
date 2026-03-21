# syntax=docker/dockerfile:1
# Frontend
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json ./
# Lockfile optional in repo
COPY web/package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi
COPY web/ ./
RUN npm run build

# Go binary (pure Go; modernc.org/sqlite)
FROM golang:1.22-alpine AS gobuild
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /tracker ./cmd/tracker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=gobuild /tracker ./tracker
COPY --from=web /web/dist ./web/dist
ENV TRACKER_DB_PATH=/data/tracker.db
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["./tracker"]
CMD ["serve", "-p", "8080", "--db", "/data/tracker.db"]
