# Clarity Auth Test Console — Docker Container

Serves `clarity-auth-test.html` via nginx on port 3000 (configurable).

---

## Quick Start

```bash
# 1. Place the HTML file in this directory (copy from the zip)
cp /path/to/clarity-auth-test.html .

# 2. Build and start
docker-compose up -d

# 3. Open in browser
open http://localhost:3000
```

---

## Requirements

- Docker 24+
- Docker Compose v2 (`docker compose` or `docker-compose`)
- The `clarity-auth-test.html` file must be present in this directory before building

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `FRONTEND_PORT` | `3000` | Host port to expose the console on |

Set in a `.env` file (copy from `.env.example`):

```bash
cp .env.example .env
# Edit FRONTEND_PORT if 3000 is already in use
```

---

## Usage

### Start in background
```bash
docker-compose up -d
```

### View logs
```bash
docker-compose logs -f frontend
```

### Stop
```bash
docker-compose down
```

### Rebuild after changing the HTML file
```bash
docker-compose build frontend
docker-compose up -d
```

### Run on a different port without editing docker-compose.yml
```bash
FRONTEND_PORT=8090 docker-compose up -d
```

---

## Runtime HTML Override

To update the HTML without rebuilding the image, mount it as a volume:

```bash
docker run -d -p 3000:80 \
  -v "$(pwd)/clarity-auth-test.html:/usr/share/nginx/html/index.html:ro" \
  clarity-frontend
```

---

## Connecting to the API

The console defaults to `http://localhost:8080/v1`. If your API runs elsewhere, change it in the **Config** tab and click **Save Configuration** — it persists to `localStorage`.

### API CORS setup

The Clarity API must allow requests from `http://localhost:3000`. In your API `.env`:

```env
FRONTEND_URL=http://localhost:3000
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

The API includes these origins automatically in development mode. Restart the API after changing `.env`.

### Running API and frontend together with Docker Compose

Uncomment the `api` service block in `docker-compose.yml`, ensure `clarity-api:latest` is built, then:

```bash
docker-compose up -d
```

---

## Folder Structure

```
clarity-frontend/
├── clarity-auth-test.html   # ← place this file here (from the zip)
├── Dockerfile
├── docker-compose.yml
├── nginx.conf
├── .env.example
└── README.md
```

---

## Health Check

```bash
curl http://localhost:3000/health
# {"status":"ok"}
```
