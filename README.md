# Time Tracker

## Используемые технологии

- Go (1.22.4)
- Postgres v14
- Make - для автоматизации управления проектом
- Docker и Docker compose - как основное средство запуска проекта
- [Migrate v4](https://github.com/golang-migrate/migrate) - для миграций базы данных
- [Swag](https://github.com/swaggo/swag) - для генерации swagger/openapi документации
- [Ogen](https://github.com/ogen-go/ogen) - для генерации swagger/openapi клиента
- JS и Express - для тестового сервиса по получению информации о пользователях

## Конфигурация

### Основное

- Основной способ настройки приложения Env Vars
- Есть только два флага командной строки:
  - `-cfg`(опционально) - путь до файла конфигурации (по умолчанию пустая строка)
  - `-prettyLog`(опционально) - отформатированные логи (по умолчанию `false`)
    - `true` при локальном запуске
    - `false` при stage (docker) запуске
- Есть три файлы конфигурации:
  - `.env` - для stage (docker) запуска
  - `.local.env` - для локального запуска
  - `.debug.env` - для отладки

### Переменные окружения

- Основные переменные окружения:
  - `HTTP_HOST` - адрес хоста (по умолчанию `localhost` или `0.0.0.0`, но есть некоторые проблемы с указанием хоста в docker из-под WSL)
  - `HTTP_PORT` - порт (по умолчанию `8080`)
  - `*` `DB_DSN` - строка подключения к базе данных, без указыния протокола (`<user>:<password>@<host>:<port>/<db>?<options>`)
  - `DB_AUTOMIGRATE` - автоматическая миграция базы данных (по умолчанию `true`)
  - `*` `PEOPLE_SERVICE_URL` - URL сервиса для получения информации о пользователях
- В файлах конфигурации можно найти дополнительные переменные, но они используются, либо для удобства, либо конфигурации других служб, к примеру docker compose

- `*` - обязательная переменная

## Запуск

### Клонирование

```bash
git clone https://github.com/protomem/time-tracker.git
cd time-tracker
```

### Docker и Docker compose(рекомендуется)

- Настройки по умолчанию
  - Приложение доступно по `localhost:8080` или `0.0.0.0:8080`
  - База данных доступна по `localhost:5432`
  - Mock People Service доступен по `localhost:8081`
  - Логи в формате JSON
  - Автоматическая миграция

```bash
make run/stage
# или
docker compose up -d

# для остановки

make stop/stage
# или
docker compose down
```

### Локальный запуск(для разработки)

- Запускает только приложение по адресу `localhost:8080` или `0.0.0.0:8080`

```bash
make run/local
# or (live reload)
make run/local/live
```

### Отдельные сервисы

- Запуск базы данных в контейнере: `make run/stage/db`
- Запуск mock-people-service в контейнере: `make run/stage/mock-people-service`
- Запуск mock-people-service в локальном режиме: `make run/local/mock-people-service`

## Миграции

- По умолчанию, включена автоматическая миграция. Но можно запустить (или откатить) ее вручную:

```bash
DB_DSN="<db_dsn>" make migrations/up # запустить миграции

DB_DSN="<db_dsn>" make migrations/down # откатить миграции
```

- Ести и другие команды для миграции:

```bash
DB_DSN="<db_dsn>" make migrations/new name=<name> # создать новую миграцию

DB_DSN="<db_dsn>" make migrations/goto version=<version> # перейти на версию миграции

DB_DSN="<db_dsn>" make migrations/force version=<version> # применить миграцию версии
```

## Mock People Service

Простая реализация [Swagger/OpenAPI спецификации](./api/external_api/people_service.yaml) c использованием JS и Express и предназначенная для тестирования приложения.

- Данные о пользователях хранятся в файле `db.js`, в виде массива объектов `components.schemas.People`.
- Имеет одну переменную окружения: `PORT`- порт, по умолчанию `3000`.

## Endpoints

- `/` или `/swagger/` - Swagger UI
- `/api/v1`
  - `/status` - статус сервиса
  - `/users`
    - `GET /` - получение всех пользователей
    - `GET /{userId}/stats` - трудозатраты пользователя
    - `POST /` - добавление пользователя
    - `PUT /{userId}` - обновление пользователя
    - `DELETE /{userId}` - удаление пользователя
  - `/sessions`
    - `GET /{userId}` - получение всех cессий пользователя
    - `POST /{userId}/{taskId}` - старт сессии
    - `DELETE /{userId}/{taskId}` - завершение сессии

## Примечания

- Для указания периода используйте формат `<год>-<месяц>-<день> <часы>:<минуты>`
  - Пример: 2024-06-02 08:03

## TODO

- Сегенерировать тестовые данные
