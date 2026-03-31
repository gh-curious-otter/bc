#!/bin/bash
set -e

# Start virtual framebuffer
Xvfb :99 -screen 0 1280x720x24 -ac &
sleep 1

# Start lightweight window manager
fluxbox -display :99 &
sleep 1

# Start VNC server (no password, listen on localhost only)
x11vnc -display :99 -forever -nopw -listen 0.0.0.0 -rfbport 5900 -shared -bg

# Start noVNC web client (proxies VNC on port 6080)
websockify --web /usr/share/novnc 6080 localhost:5900 &

echo "noVNC running at http://localhost:6080/vnc.html"
echo "Starting Playwright MCP server on port 3000..."

# Start Playwright MCP server with SSE transport on port 3000
# Headed is the default when DISPLAY is set, so browsers are visible
exec npx @playwright/mcp@latest --port 3000 --host 0.0.0.0 --allowed-hosts '*' --browser chromium
