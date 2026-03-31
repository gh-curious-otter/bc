FROM ubuntu:24.04

# Install display, VNC, noVNC, and Node.js
RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb \
    x11vnc \
    fluxbox \
    novnc \
    websockify \
    curl \
    ca-certificates \
    && curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install Playwright MCP server, then install Chromium + deps via its own Playwright
RUN npm install -g @playwright/mcp@latest \
    && npx playwright install --with-deps chromium

# Environment
ENV DISPLAY=:99
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
ENV NODE_ENV=production

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 6080 3000

ENTRYPOINT ["/entrypoint.sh"]
