# Деплой на VPS (Docker)

Стек: PostgreSQL, Go API, mediasoup signaling, Next.js, Nginx на порту 80.

Папка **`deploy/`** находится в репозитории **backend**; на сервере рядом должны лежать клоны **frontend** и **signaling** (см. раздел 3).

## 1. Сервер

- Ubuntu 22.04/24.04 (или другой Linux с Docker).
- Открыть в фаерволе: **TCP 80**, **UDP 40000–41000** (WebRTC RTP для mediasoup).

Пример `ufw`:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 40000:41000/udp
sudo ufw enable
```

## 2. Docker

```bash
sudo apt update && sudo apt install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt update && sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
```

## 3. Код на VPS

В одном каталоге (например `~/DIPLOM`) три репозитория **с такими именами папок**:

```text
DIPLOM/
├── video-conference-app-backend/   ← внутри него папка deploy/
├── video-conference-app-frontend/
└── video-conference-app-signaling/
```

## 4. Переменные окружения

```bash
cd ~/DIPLOM/video-conference-app-backend/deploy
cp .env.example .env
nano .env   # или vim
```

Обязательно:

| Переменная | Значение |
|------------|----------|
| `VPS_PUBLIC_IP` | Публичный IPv4 этой машины (`curl -4s ifconfig.me`). |
| `POSTGRES_PASSWORD`, `JWT_SECRET` | Случайные длинные строки. |
| `PUBLIC_BASE_URL` | Как браузер ходит к сайту, **без** завершающего `/`, например `https://meet.example.com` или `http://IP`. |
| `PUBLIC_WS_URL` | Тот же хост, схема `ws`/`wss` и путь `/ws`, например `wss://meet.example.com/ws`. |
| `ALLOWED_ORIGIN_SUFFIXES` | Для CORS: суффикс Origin (часто совпадает с доменом из URL, без `https://`). Несколько значений — через запятую. |

После **любого** изменения `PUBLIC_*` нужно пересобрать фронт:

```bash
docker compose -f docker-compose.prod.yml --env-file .env build web --no-cache
docker compose -f docker-compose.prod.yml --env-file .env up -d
```

## 5. Запуск

```bash
cd ~/DIPLOM/video-conference-app-backend/deploy
docker compose -f docker-compose.prod.yml --env-file .env up -d --build
```

Проверка: откройте в браузере `PUBLIC_BASE_URL`.

## 6. HTTPS (рекомендуется)

Сейчас Nginx слушает только **порт 80**. Для cookies и камеры в проде лучше TLS.

Варианты:

- Поставить **Caddy** перед контейнерами и проксировать на `localhost:80` с автоматическим Let’s Encrypt.
- Или **Certbot** + правки `nginx/prod.conf` (`listen 443 ssl`, пути к сертификатам) и проброс `443:443` у сервиса `nginx`.

После включения HTTPS обновите в `.env` схемы на `https://` и `wss://`, пересоберите `web` (см. выше).

## 7. Ограничения по умолчанию

Диапазон RTP в compose: **40000–41000** UDP (~1000 портов). Для большего числа участников расширьте диапазон в `docker-compose.prod.yml` (и `MEDIASOUP_RTC_*` у `signaling`) и откройте те же порты в фаерволе.
