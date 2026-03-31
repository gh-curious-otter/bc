#!/bin/bash
set -e

# Clear stale X11 locks
rm -f /tmp/.X*-lock /tmp/.X11-unix/X* 2>/dev/null

# Start Xvfb (virtual framebuffer)
Xvfb :99 -screen 0 1280x720x24 &
export DISPLAY=:99

# Start x11vnc (VNC server on the virtual display)
x11vnc -display :99 -forever -nopw -shared -rfbport 5900 &

# Start noVNC (web-based VNC client on port 6080)
websockify --web=/usr/share/novnc 6080 localhost:5900 &

# Start Playwright MCP server
exec npx -y @playwright/mcp@latest --port 3000
