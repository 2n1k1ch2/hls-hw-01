.PHONY: all down logs test_all test_down build

# Запустить весь проект
all:
	docker compose up -d --build

# Остановить основной проект
down:
	docker compose down

# Запустить тестовую БД и тесты
test_all:
	docker compose -f docker-compose.test.yaml up -d --wait &&
	TEST_DATABASE_URL="host=localhost port=5432 dbname=pqgo_test user=postgres password=test sslmode=disable" go test ./... -v

# Остановить тестовую БД
test_down:
	docker compose -f docker-compose.test.yaml down

# Собрать  локально
build:
	go build -o bin/server .