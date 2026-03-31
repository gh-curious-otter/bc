#!/bin/sh
# Clear stale X11 display locks from previous crashes
rm -f /tmp/.X*-lock /tmp/.X11-unix/X* 2>/dev/null

# Execute the original entrypoint
exec /entrypoint.sh "$@"
