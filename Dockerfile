FROM golang:1.26-alpine AS builder

LABEL maintainer="yokko"
LABEL app="site-monitor"

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum ./

# этот слой закешируется если все уже было установлено
RUN go mod download

COPY . .

# отключаем cgo, указываем target OS, отключаем отладочную инфу (существенно уменьшает размер)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s -X main.buildVersion=v1.0.0-docker" -o site-monitor ./cmd/monitor

# абсолютно пустой образ (0 мб)
FROM scratch

WORKDIR /app

# копируем ssl сертификаты и таймзоны из билдера (без них запросы чекинга сайтов будут падать, а даты в логах будут некорректными)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /build/site-monitor .

COPY --from=builder /build/configs/sites.yaml ./configs/sites.yaml

EXPOSE 8080

CMD ["./site-monitor", "-config", "./configs/sites.yaml"]