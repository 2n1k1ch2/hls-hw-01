# ITMO-DDIA-2026-MSE/MHS-HW-1

## Пререквизиты

- Docker и Docker Compose
- Go 1.27.2
- Создать `.env` на основе `.env.example` и заполнить необходимые переменные окружения.

## Запуск проекта

Запустить все
```bash
make all
```

После запуска сервисы доступны по адресам:

- **API:** http://localhost:8080
- **VictoriaMetrics:** http://localhost:8428
- **Grafana:** http://localhost:3000/d/shop-api-auto

## Тестирование

Запустить тестовую базу данных и тесты микросервиса:

```bash
make test_all
```

Остановить тестовую базу данных:

```bash
make test_down
```
Остановить основной проект:
```bash
make down
```

Собрать Go-приложение локально:
```bash
make build
```