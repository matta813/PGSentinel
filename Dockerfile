# syntax=docker/dockerfile:1.26
FROM node:24-alpine AS frontend
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X gitlab.scruzzi.com/root/postgresqlui/internal/buildinfo.Version=${VERSION} -X gitlab.scruzzi.com/root/postgresqlui/internal/buildinfo.CommitSHA=${COMMIT_SHA} -X gitlab.scruzzi.com/root/postgresqlui/internal/buildinfo.BuildTime=${BUILD_TIME}" -o /out/pgsentinel ./cmd/pgsentinel

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && addgroup -S -g 10001 pgsentinel && adduser -S -D -H -u 10001 -G pgsentinel pgsentinel
WORKDIR /app
COPY --from=backend /out/pgsentinel /usr/local/bin/pgsentinel
COPY --from=frontend /src/frontend/dist /app/web
RUN mkdir /data && chown pgsentinel:pgsentinel /data
USER pgsentinel
ENV PGSENTINEL_DATA_DIR=/data PGSENTINEL_FRONTEND_DIR=/app/web PGSENTINEL_LISTEN_ADDR=:8080
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1
ENTRYPOINT ["pgsentinel"]
