include .env
export

.PHONY: infra-start infra-stop migrate migrate-create migrate-downrun generate

infra-start: ## Запустить инфраструктуру
	tripgoctl environment start
	# перейти в WD для WSL: cd /mnt/c/Users/89307/GolandProjects/trip-go

infra-stop: ## Остановить инфраструктуру
	tripgoctl environment stop

migrate-create: ## Создать SQL-миграцию с порядковым номером
	go tool goose -s -dir migrations create "$(NAME)" sql

migrate: ## Применить миграции к БД
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down: ## Откатить последнюю миграцию
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status: ## Статус миграций
	go tool goose -dir migrations postgres "$(DATABASE_URL)" status

run: ## Запустить сервис локально
	go run cmd/trip-service/main.go

generate: generate-openapi ## Все генерации

generate-openapi: ## Сгенерировать типы и интерфейсы из OpenAPI
	go tool oapi-codegen \
      -generate types,chi-server \
      -package api \
      -o internal/generated/api.gen.go \
      contracts/openapi/trip-service.openapi.yaml
