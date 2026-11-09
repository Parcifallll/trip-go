# Лабораторная работа 1. HTTP API и PostgreSQL


## Запуск

```bash
tripgoctl cluster start        # один раз на машине
tripgoctl environment start    # поднять PostgreSQL
tripgoctl connect              # адреса компонентов, формирует .env (шаблон: .env.example)
make migrate                   # применить миграции
make run                       # запустить сервис на HTTP_ADDR
```

Проверка:

```bash
curl -i localhost:8080/health
curl -i localhost:8080/ready
```

Остальные команды: `make migrate-down`, `make migrate-status`, `make generate` (код из OpenAPI).

## Переменные окружения

| Переменная | Назначение |
|---|---|
| `HTTP_ADDR` | адрес HTTP-сервера |
| `LOG_LEVEL` | уровень логирования |
| `SHUTDOWN_TIMEOUT` | таймаут graceful shutdown |
| `HTTP_READ_TIMEOUT` | таймаут чтения запроса |
| `HTTP_READ_HEADER_TIMEOUT` | таймаут чтения заголовков |
| `HTTP_WRITE_TIMEOUT` | таймаут записи ответа |
| `HTTP_IDLE_TIMEOUT` | таймаут простоя соединения |
| `DATABASE_URL` | строка подключения к PostgreSQL |
| `DATABASE_MAX_CONNS` | максимум соединений в пуле |
| `DATABASE_MIN_CONNS` | минимум соединений в пуле |
| `DATABASE_MAX_CONN_LIFETIME` | максимальное время жизни соединения |
| `DATABASE_CONNECT_TIMEOUT` | таймаут подключения к БД |
| `DATABASE_QUERY_TIMEOUT` | таймаут запроса к БД |

Все переменные обязательные, `.env` обязателен.

## Что сделано

Ручки:

Контракт в`contracts/openapi/trip-service.openapi.yaml`.

`POST /api/v1/trips`

`GET /api/v1/trips/{tripId}`

`POST /api/v1/trips/{tripId}/finish`

`/health` и `/ready`.

## Решения

**Уровень изоляции: `READ COMMITTED`.** Задаётся при старте транзакции в `Do` 

**Обоснование:** Все операции сервиса затрагивают одну строку `trips` по ключу, поэтому более строгая изоляция не нужна: от гонок защищают условный `UPDATE` и уникальный индекс, а не уровень изоляции.
`SERIALIZABLE` потребовал бы повторов при конфликтах сериализации без выигрыша в корректности.


**Менеджер транзакций:** `Do(ctx, fn)` открывает транзакцию и кладёт её в `context`, репозитории достают её из контекста и работают в ней (иначе используют пул).

Создание поездки идёт в одном `Do`: `INSERT` в `trips` и `INSERT` в `trip_status_history`,при ошибке второй вставки в `trips` ничего не остаётся.

**Запрет двух активных поездок у водителя:** Частичный уникальный индекс `trips_driver_active_idx` на `trips (driver_id) WHERE status = 'active'`.
Проверка идёт на уровне БД, поэтому она атомарна и работает при конкурентных запросах. 

Нарушение индекса (код `23505`) превращается в доменную ошибку «водитель занят» и в ответ `409 driver_busy`.

**Завершение поездки:** Атомарный `UPDATE ... WHERE id = $1 AND status = 'active'`. Если обновлено 0 строк, дополнительная проверка существования разделяет `404 trip_not_found` и `409 trip_completed`.