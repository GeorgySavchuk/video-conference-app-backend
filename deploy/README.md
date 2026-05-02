# Деплой на VPS (Docker)

Стек: PostgreSQL, Go API, mediasoup signaling, Next.js, Nginx на порту 80.

Папка **`deploy/`** находится в репозитории **backend**; на сервере рядом должны лежать клоны **frontend** и **signaling** (см. раздел 3).

## 1. Сервер

- Ubuntu 22.04/24.04 (или другой Linux с Docker).
- Открыть в фаерволе: **TCP 80** (и **TCP 443** для HTTPS), **UDP 40000–41000** (WebRTC RTP для mediasoup).

Пример `ufw`:

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 40000:41000/udp
sudo ufw enable
```

В панели облака (Yandex и т.д.) добавьте те же правила для группы безопасности ВМ.

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

## 6. HTTPS (Let’s Encrypt + Certbot на хосте)

Браузер даёт камеру/микрофон только на **https://** (или localhost). Ниже — вариант **бесплатного** сертификата **Let’s Encrypt** через **webroot**: Nginx в Docker отдаёт `/.well-known/acme-challenge/`, Certbot пишет файлы в `./certbot/www` на хосте.

### 6.1. Домен

1. Зарегистрируйте домен (или поддомен), например `meet.example.com`.
2. В DNS создайте **A-запись** на **публичный IPv4** этой ВМ (как в `VPS_PUBLIC_IP`).
3. Подождите, пока `dig +short meet.example.com` с вашего ПК покажет этот IP.

### 6.2. Фаервол

Откройте **TCP 80** и **TCP 443** (см. раздел 1). Без 80 Let’s Encrypt не выпустит сертификат (проверка HTTP-01).

### 6.3. Первый запуск только по HTTP

Убедитесь, что стек поднят (порт 80 отвечает):

```bash
cd ~/DIPLOM/video-conference-app-backend/deploy
docker compose -f docker-compose.prod.yml --env-file .env up -d
```

### 6.4. Certbot на ВМ (на хосте, не в контейнере)

```bash
sudo apt update && sudo apt install -y certbot
```

Выпустить сертификат (**подставьте свой домен** и почту):

```bash
cd ~/DIPLOM/video-conference-app-backend/deploy
sudo certbot certonly --webroot \
  -w "$(pwd)/certbot/www" \
  -d meet.example.com \
  --email you@example.com \
  --agree-tos \
  --non-interactive
```

Если Certbot пишет про недоступность challenge — проверьте DNS, фаервол и что контейнер **nginx** запущен и `./certbot/www` смонтирован (уже есть в `docker-compose.prod.yml`).

### 6.5. Конфиг HTTPS в Nginx

```bash
cd ~/DIPLOM/video-conference-app-backend/deploy/nginx
cp prod-https.conf.example prod-https.conf
sed -i 's/YOUR_DOMAIN/meet.example.com/g' prod-https.conf
```

(или отредактируйте `prod-https.conf` вручную: `server_name` и пути `/etc/letsencrypt/live/…` должны совпадать с доменом из Certbot.)

### 6.6. Поднять Nginx с 443

Сертификаты лежат на хосте в `/etc/letsencrypt`; второй compose-файл монтирует их в контейнер:

```bash
cd ~/DIPLOM/video-conference-app-backend/deploy
docker compose -f docker-compose.prod.yml -f docker-compose.https.yml --env-file .env up -d
```

### 6.7. Обновить `.env` и пересобрать фронт

В `.env` задайте **https** и **wss** (тот же домен):

```env
PUBLIC_BASE_URL=https://meet.example.com
PUBLIC_WS_URL=wss://meet.example.com/ws
ALLOWED_ORIGIN_SUFFIXES=meet.example.com
```

Пересборка фронта (обязательно — в образ зашиваются `NEXT_PUBLIC_*`):

```bash
docker compose -f docker-compose.prod.yml -f docker-compose.https.yml --env-file .env build web --no-cache
docker compose -f docker-compose.prod.yml -f docker-compose.https.yml --env-file .env up -d
```

Проверка: откройте `https://meet.example.com` — замочек в браузере, камера может запросить разрешение.

### 6.8. Продление сертификата

Certbot ставит cron/timer на хосте. После продления достаточно перезагрузить nginx в Docker:

```bash
docker compose -f docker-compose.prod.yml -f docker-compose.https.yml --env-file .env exec nginx nginx -s reload
```

## 7. Ограничения по умолчанию

Диапазон RTP в compose: **40000–41000** UDP (~1000 портов). Для большего числа участников расширьте диапазон в `docker-compose.prod.yml` (и `MEDIASOUP_RTC_*` у `signaling`) и откройте те же порты в фаерволе.
