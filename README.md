## Site monitor

### Назначение сервиса
Данный сервис предназначен для мониторинга работы и доступности сайтов по адресам, указанным в конфигурации

### Инструкции по работе с программой

- Сборка
    ```bash
      go build ./...
    ```
- Запуск
    ```bash
      go run cmd/monitor/main.go
    ```

- Проверка кода линтером
    ```bash
      golangci-lint run
    ```
  
### Установка линтера

- MacOS
    ```bash
      brew install golangci/tap/golangci-lint
    ```
- Arch Linux
    ```bash
      yay -S golangci-lint
    ```
    ```bash
      sudo pacman -S golangci-lint
    ```

### Swagger и автогенерация документации

1. Установка генератора (CLI-утилита):

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

2. Генерация (в корне проекта)

```bash
swag init -g cmd/monitor/main.go
```

### Контейнеризация (Docker)

1. Сборка образа (запускать в корне)

```bash
docker build -t site-monitor .
```

2. Проверка размера образа

```bash
docker images | grep site-monitor
```

3. Запуск контейнера

```bash
docker run -p 8080:8080 site-monitor
```

### Docker-compose

1. Запуск и сборка

```bash
docker compose up --build -d
```

2. Проверка контейнеров

```bash
docker ps
```

### Управление данными (Docker volumes)

База данных PostgreSQL в докере использует Volumes для персистентного хранения данных. Это значит что данные не потеряются при перезапуске или падении контейнера

### Полезные команды для работы с докером

1. Остановить проект (данные БД сохранятся)

```bash
docker compose down
```

2. Остановить проект и полностью удалить данные (volumes)

```bash
docker compose down -v
```

3. Посмотреть список всех volumes в системе

```bash
docker volumes ls
```

4. Перезапустить приложение (нужно к примеру если мы во время работы приложения отредактировали configs/sites.yaml и хотим чтоб оно обновилось)

```bash
docker restart monitor-app
```