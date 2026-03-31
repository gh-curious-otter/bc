#!/bin/bash
set -e

# Clear stale X11 locks
rm -f /tmp/.X*-lock /tmp/.X11-unix/X* 2>/dev/null

# Virtual display for visible browser
Xvfb :99 -screen 0 1280x720x24 &
export DISPLAY=:99

# VNC server + noVNC web client
x11vnc -display :99 -forever -nopw -shared -rfbport 5900 2>/dev/null &
websockify --web=/usr/share/novnc 6080 localhost:5900 2>/dev/null &

# Shared volume for screenshots
mkdir -p /tmp/bc-shared
chmod 777 /tmp/bc-shared
export PLAYWRIGHT_OUTPUT_DIR=/tmp/bc-shared

# Playwright MCP server (headed — visible in VNC)
# --host 0.0.0.0     : accept external connections
# --allowed-hosts '*' : accept any Host header (Docker networking)
# --port 3000         : SSE transport port
# --no-sandbox        : required in Docker
exec npx -y @playwright/mcp \
  --browser chromium \
  --host 0.0.0.0 \
  --port 3000 \
  --allowed-hosts '*' \
  --no-sandbox
