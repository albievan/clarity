# ── Clarity Auth Test Console — Frontend Container ───────────────────────────
#
# Serves the single-file auth test console (clarity-auth-test.html) via nginx.
# The HTML file is renamed to index.html inside the container so it is served
# at the root URL without any path suffix.
#
# Build:
#   docker build -t clarity-frontend .
#
# Run (standalone):
#   docker run -d -p 3000:80 --name clarity-frontend clarity-frontend
#
# Run (with docker-compose — see docker-compose.yml):
#   docker-compose up -d
#
# Mount a custom HTML file at runtime instead of the built-in one:
#   docker run -d -p 3000:80 \
#     -v /path/to/your/clarity-auth-test.html:/usr/share/nginx/html/index.html:ro \
#     clarity-frontend
#
FROM nginx:1.27-alpine

# Remove default nginx site
RUN rm -f /etc/nginx/conf.d/default.conf

# Copy our nginx site config
COPY nginx.conf /etc/nginx/conf.d/clarity.conf

# Copy the test console HTML — renamed to index.html so it serves at /
COPY clarity-auth-test.html /usr/share/nginx/html/index.html

# nginx runs as the default user; no privilege escalation needed
EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost/health || exit 1

CMD ["nginx", "-g", "daemon off;"]
