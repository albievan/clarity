# Clarity — Marketing Website

The public-facing marketing website for the Clarity Budget Management Platform.

## Stack

- **Static HTML/CSS/JS** — no build step required
- **Nginx 1.25 Alpine** — minimal, production-grade web server
- **Docker** — single container deployment

## Quick Start

### Docker Compose (recommended)

```bash
docker compose up -d
```

Site will be available at **http://localhost:8080**

### Docker only

```bash
# Build
docker build -t clarity-website .

# Run
docker run -d -p 8080:80 --name clarity-website clarity-website
```

### Local development (no Docker)

Open `public/index.html` directly in a browser, or use any static file server:

```bash
npx serve public
# or
python3 -m http.server 8080 --directory public
```

## Project Structure

```
clarity-website/
├── Dockerfile
├── docker-compose.yml
├── nginx.conf
├── README.md
└── public/
    ├── index.html
    └── assets/
        ├── style.css
        ├── main.js
        ├── logo.svg          # Full logo (dark backgrounds)
        ├── logo-white.svg    # Full logo (light backgrounds)
        └── logo-mark.svg     # Icon only (favicon, small sizes)
```

## Deployment

The container exposes port 80 internally. Map to any host port.

**Production with reverse proxy (e.g. Traefik or Nginx upstream):**

```bash
docker run -d \
  --name clarity-website \
  --network your-proxy-network \
  -e VIRTUAL_HOST=clarity.yourcompany.com \
  clarity-website
```

**Health check endpoint:** `GET /health` → `200 ok`
