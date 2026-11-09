include .env
export

.PHONY: infra-start infra-stop migrate migrate-create migrate-downrun generate

infra-start: ## Запустить инфраструктуру
	tripgoctl environment start
	# перейти в WD для WSL: cd /mnt/c/Users/89307/GolandProjects/trip-go

infra-stop: ## Остановить инфраструктуру
	tripgoctl environment stop

cluster-start: ## один раз на машине
	tripgoctl cluster start

cluster-connect:  ## адреса и ссылки на веб-интерфейсы
	tripgoctl connect

get-pods:
	kubectl get pods -n tripgo-lab-02

migrate-create: ## Создать SQL-миграцию с порядковым номером
	go tool goose -s -dir migrations create "$(NAME)" sql

migrate: ## Применить миграции к БД
	go tool goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down: ## Откатить последнюю миграцию
	go tool goose -dir migrations postgres "$(DATABASE_URL)" down

migrate-status: ## Статус миграций
	go tool goose -dir migrations postgres "$(DATABASE_URL)" status

run: ## Запустить сервис локально
	go run cmd/service/main.go

generate: generate-openapi generate-mockgen ## Все генерации

generate-openapi: ## Сгенерировать типы и интерфейсы из OpenAPI
	go tool oapi-codegen \
      -generate types,chi-server \
      -package api \
      -o internal/generated/api.gen.go \
      contracts/openapi/trip-service.openapi.yaml

generate-mockgen:
	go generate ./...

test:
	go test -race -count=1 -coverpkg=./internal/usecase/...,./internal/domain/... -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

test-integration:
	go test -race -count=1 -tags=integration ./...

lint:
	golangci-lint run