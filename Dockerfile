# syntax=docker/dockerfile:1.26@sha256:ecfaec9ed6d810b56388c508f4121597bfbba70d41a6dfeee4d8cad5f295fc32
FROM node:24-alpine@sha256:d32cdf619f63fe0471182d08996dd516c6275bb5fd31ae06e55a570bd9e1ad43 AS frontend
WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine@sha256:70b46548e42db77e0966aaf3619fd068734dc6c77584d526b91126504fd95816 AS backend
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
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X github.com/matta813/pgsentinel/internal/buildinfo.Version=${VERSION} -X github.com/matta813/pgsentinel/internal/buildinfo.CommitSHA=${COMMIT_SHA} -X github.com/matta813/pgsentinel/internal/buildinfo.BuildTime=${BUILD_TIME}" -o /out/pgsentinel ./cmd/pgsentinel

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
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
