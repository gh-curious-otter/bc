#!/usr/bin/env bash
# Test how to send "submit" to Cursor agent so it runs the command instead of inserting newline.
# Run this, then attach with: tmux attach -t bc-test-enter
# Then run ONE of the "Try" commands below and see if Cursor runs the line or just adds newline.

set -e
SESSION="bc-test-enter"

# Use cursor-agent if available, else cursor (from config)
CMD="${CURSOR_AGENT_CMD:-cursor --dangerously-skip-permissions}"
if command -v cursor-agent &>/dev/null; then
  CMD="cursor-agent"
fi

echo "Using agent command: $CMD"
echo ""

if tmux has-session -t "$SESSION" 2>/dev/null; then
  echo "Session $SESSION already exists. Kill it with: tmux kill-session -t $SESSION"
  echo "Then re-run this script to start fresh."
else
  echo "Creating tmux session $SESSION with $CMD..."
  tmux new-session -d -s "$SESSION" -c "$(pwd)" -- "$CMD"
  echo "Session created."
fi

echo ""
echo "Attach to see the Cursor UI:  tmux attach -t $SESSION"
echo ""
echo "In another terminal, send a test line using ONE of these and watch Cursor:"
echo ""
echo "  Method A - tmux key name Enter (what bc currently uses):"
echo "    tmux send-keys -t $SESSION 'bc report working' Enter"
echo ""
echo "  Method B - Literal carriage return (\\r):"
echo "    tmux send-keys -t $SESSION -l \$'bc report working\\r'"
echo ""
echo "  Method C - Literal newline (\\n):"
echo "    tmux send-keys -t $SESSION -l \$'bc report working\\n'"
echo ""
echo "Whichever method makes Cursor *run* the command (not just insert newline) is the one to use."
echo "When done testing:  tmux kill-session -t $SESSION"
