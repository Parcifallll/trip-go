# trip-go

## Требования

- Go (версия из `go.mod`)
- Docker и `tripgoctl`
- goose

## Запуск

Перед запуском нужно убедиться, что в `.env` заданы все переменные (шаблон — `.env.example`).
Если обязательные переменные отсутствуют, сервис не запустится.

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect

make migrate    # накатить миграции
make run        # запустить сервис
```

## Переменные окружения

| Переменная | Назначение |
|---|---|
| `HTTP_ADDR` | адрес HTTP-сервера |
| `DATABASE_URL` | строка подключения к PostgreSQL |
| `DATABASE_MAX_CONNS` | максимум соединений в пуле |
| `DATABASE_MIN_CONNS` | минимум соединений в пуле |
| `DATABASE_MAX_CONN_LIFETIME` | время жизни соединения |
| `DATABASE_CONNECT_TIMEOUT` | таймаут подключения |
| `DATABASE_QUERY_TIMEOUT` | таймаут запросов |
| `LOG_LEVEL` | уровень логов |
| `SHUTDOWN_TIMEOUT` | таймаут graceful shutdown |


## Что сделано

- структура репозитория по `conventions.md`, конфигурация из env
- HTTP-сервер на `chi` с таймаутами, `/health`, `/ready`, graceful shutdown по `SIGINT`/`SIGTERM`
- типы и серверные интерфейсы сгенерированы из OpenAPI
- миграции goose
- `pgxpool`, репозитории на `pgx` + `squirrel`, менеджер транзакций
- три ручки с ошибками в `problem+json`

## Решения

### Уровень изоляции

Read Committed (задан явно в `Do`). Оба инварианта закреплены в БД: частичным
уникальным индексом и условием в `UPDATE`, поэтому более строгий уровень пользы не
даёт. Serializable при конфликтах возвращал бы 500 вместо 409 и требовал бы ретраев.

### Менеджер транзакций

`Do(ctx, fn)` открывает транзакцию, кладёт в `context` и вызывает `fn`. При `nil`
делается `COMMIT`, при ошибке или панике — `ROLLBACK`. Репозиторий берёт исполнителя из
контекста, если есть транзакция то работает через неё, иначе через пул. Вложенный `Do`
переиспользует существующую транзакцию.

### Запрет двух активных поездок

Гарантию дает частичный уникальный индекс `trips_driver_active_idx ON trips (driver_id) WHERE status = 'active'`.
