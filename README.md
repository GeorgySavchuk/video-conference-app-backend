# Video Conference App Backend

## Описание
Это бэкенд-приложение для видеоконференций, разработанное на Go с использованием фреймворка Gin и базы данных PostgreSQL. Приложение предоставляет API для управления встречами и пользователями.

## Установка

### Предварительные требования
- Go (версия 1.21 или выше)
- PostgreSQL (версия 12 или выше)
- Docker (опционально, для использования Docker Compose)

### Клонирование репозитория
```bash
git clone https://github.com/GeorgySavchuk/video-conference-app-backend.git
cd video-conference-app-backend
```

### Деплой на VPS (Docker)

Каталог **`deploy/`** в этом репозитории: `docker-compose.prod.yml`, Nginx и шаблон `.env` для поднятия API, Postgres, фронта и signaling на одной машине. Рядом на диске должны лежать клоны **video-conference-app-frontend** и **video-conference-app-signaling** — см. `deploy/README.md`.

### Настройка базы данных
1. Создайте базу данных PostgreSQL:
   ```sql
   CREATE DATABASE video_conference;
   ```

2. Скопируйте `.env.example` в `.env` в корне бэкенда и заполните значения. При `go run main.go` переменные подхватываются из `.env` автоматически.
   Минимум для БД:
   ```
   DB_HOST=localhost
   DB_USER=postgres
   DB_PASSWORD=your_password
   DB_NAME=video_conference
   DB_PORT=5432
   JWT_SECRET=your_jwt_secret
   ```
   Для отправки писем (приглашения, напоминания) добавьте `SMTP_HOST`, `SMTP_FROM` и при необходимости `SMTP_USER`, `SMTP_PASSWORD` (см. комментарии в `.env.example`).

### Установка зависимостей
```bash
go mod download
```

### Запуск приложения
```bash
go run main.go
```

### Использование Docker
Если вы предпочитаете использовать Docker, вы можете запустить приложение с помощью Docker Compose:
```bash
docker-compose up --build
```

## API
### Регистрация пользователя
- **POST** `/auth/signup`
- **Тело запроса**:
  ```json
  {
    "name": "username",
    "password": "password"
  }
  ```

### Авторизация пользователя
- **POST** `/auth/signin`
- **Тело запроса**:
  ```json
  {
    "name": "username",
    "password": "password"
  }
  ```

### Управление встречами
- **Создание встречи**: `POST /meetings`
- **Получение всех встреч**: `GET /meetings`
- **Обновление встречи**: `PUT /meetings/:id`
- **Удаление встречи**: `DELETE /meetings/:id`