# ── BUILD STAGE ──────────────────────────────────────────
# Using alpine-based nginx for minimal image size
FROM nginx:1.25-alpine

LABEL maintainer="Clarity <albievan@gmail.com>"
LABEL description="Clarity Budget Management Platform — Marketing Website"
LABEL version="1.0.0"

# Remove default nginx static assets and config
RUN rm -rf /usr/share/nginx/html/* && \
    rm /etc/nginx/conf.d/default.conf

# Copy custom nginx configuration
COPY nginx.conf /etc/nginx/conf.d/clarity.conf

# Copy static website files
COPY public/ /usr/share/nginx/html/

# Set correct permissions
RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chmod -R 755 /usr/share/nginx/html

# Expose port 80
EXPOSE 80

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost/health || exit 1

CMD ["nginx", "-g", "daemon off;"]
