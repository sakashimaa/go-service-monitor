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