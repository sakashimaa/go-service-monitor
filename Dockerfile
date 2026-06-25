FROM golang:1.26-alpine AS builder
ARG VERSION=v1.0.0-docker

WORKDIR /build

COPY go.mod go.sum ./

# этот слой закешируется если все уже было установлено
RUN go mod download

COPY . .

# отключаем cgo, указываем target OS, отключаем отладочную инфу (существенно уменьшает размер)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -X main.buildVersion=${VERSION}" -o site-monitor ./cmd/monitor

FROM alpine:3.20

LABEL maintainer="yokko"
LABEL app="site-monitor"

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/site-monitor .

# ВНИМАНИЕ: Данный COPY дефолтного конфига перекрывается в docker-compose.yml
# через volume (./configs:/app/configs:ro) инструкция оставлена намеренно,
# чтобы собранный образ оставался самодостаточным и мог успешно запускаться
# в standalone-режиме (например, через обычный `docker run`) без compose.
COPY --from=builder /build/configs/sites.yaml ./configs/sites.yaml

EXPOSE 8080

CMD ["./site-monitor", "-config", "./configs/sites.yaml"]