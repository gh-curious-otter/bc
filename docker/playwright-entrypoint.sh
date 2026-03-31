#!/bin/bash
set -e

# Clear stale X11 locks
rm -f /tmp/.X*-lock /tmp/.X11-unix/X* 2>/dev/null

# Start virtual display
Xvfb :99 -screen 0 1280x720x24 &
export DISPLAY=:99

# Start VNC for optional browser viewing
x11vnc -display :99 -forever -nopw -shared -rfbport 5900 2>/dev/null &
websockify --web=/usr/share/novnc 6080 localhost:5900 2>/dev/null &

# Start Playwright MCP server
# --host 0.0.0.0      : accept connections from outside container
# --allowed-hosts '*'  : accept any Host header (Docker port mapping)
# --port 3000          : MCP server port (mapped to 3100 externally)
# --browser chromium   : use chromium (pre-installed in base image)
exec npx -y @playwright/mcp --host 0.0.0.0 --port 3000 --allowed-hosts '*' --browser chromium
