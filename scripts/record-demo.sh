#!/bin/bash
#
# Demo recording script for README GIF
# Usage: ./scripts/record-demo.sh
#
# Prerequisites:
#   - asciinema: brew install asciinema
#   - svg-term-cli: npm install -g svg-term-cli
#   - A clean test directory
#
# This script creates a simulated bc demo session for recording.
# Run with asciinema: asciinema rec demo.cast -c './scripts/record-demo.sh'
#

set -e

# Colors for simulated output
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Typing simulation speed (seconds between characters)
TYPE_DELAY=0.03
# Pause between commands
CMD_PAUSE=1.5
# Pause for hero moments
HERO_PAUSE=3

# Simulate typing a command
type_cmd() {
    echo -n "$ "
    for ((i=0; i<${#1}; i++)); do
        echo -n "${1:$i:1}"
        sleep $TYPE_DELAY
    done
    echo
    sleep 0.3
}

# Pause with message (for recording timing)
pause() {
    sleep "$1"
}

# Create temp workspace
DEMO_DIR=$(mktemp -d)
cd "$DEMO_DIR"

echo -e "${CYAN}bc Demo Recording${NC}"
echo "=================="
echo
pause 1

# Scene 1: Initialize workspace (0-5s)
type_cmd "bc init"
echo -e "${GREEN}✓${NC} Initialized bc workspace"
echo "  Created .bc/ directory"
echo "  Default config: config.toml"
pause $CMD_PAUSE

# Scene 2: Start root agent (5-10s)
type_cmd "bc up"
echo -e "${GREEN}✓${NC} Started root agent in tmux session"
echo "  Session: bc-root"
echo "  Role: root"
pause $CMD_PAUSE

# Scene 3: Create engineer agents (10-18s)
type_cmd "bc agent create --role engineer"
echo -e "${GREEN}✓${NC} Created agent: swift-falcon"
echo "  Role: engineer"
echo "  Worktree: .bc/worktrees/swift-falcon"
pause $CMD_PAUSE

type_cmd "bc agent create --role engineer"
echo -e "${GREEN}✓${NC} Created agent: bold-hawk"
echo "  Role: engineer"
echo "  Worktree: .bc/worktrees/bold-hawk"
pause $CMD_PAUSE

# Scene 4: Show status (hero moment)
type_cmd "bc status"
echo
echo -e "${CYAN}Agents${NC}"
echo "┌──────────────┬──────────┬─────────┬──────────┐"
echo "│ NAME         │ ROLE     │ STATUS  │ COST     │"
echo "├──────────────┼──────────┼─────────┼──────────┤"
echo "│ root         │ root     │ active  │ \$0.12    │"
echo "│ swift-falcon │ engineer │ active  │ \$0.00    │"
echo "│ bold-hawk    │ engineer │ active  │ \$0.00    │"
echo "└──────────────┴──────────┴─────────┴──────────┘"
echo
pause $HERO_PAUSE

# Scene 5: Open TUI dashboard (18-25s) - Hero moment
type_cmd "bc home"
echo
echo -e "${CYAN}Opening TUI dashboard...${NC}"
echo
# Simulated TUI frame
echo "┌─────────────────────────────────────────────────────────────────────┐"
echo "│  ${CYAN}bc${NC} Dashboard                                     swift-falcon ▸  │"
echo "├─────────────────────────────────────────────────────────────────────┤"
echo "│                                                                     │"
echo "│  ${CYAN}Agents${NC} (3)                    ${CYAN}Activity${NC}                          │"
echo "│  ┌─────────────────────────┐   ┌─────────────────────────────────┐ │"
echo "│  │ ▸ root         active   │   │ 10:23 root     Working on task  │ │"
echo "│  │   swift-falcon active   │   │ 10:22 falcon   Created worktree │ │"
echo "│  │   bold-hawk    active   │   │ 10:21 hawk     Initialized      │ │"
echo "│  └─────────────────────────┘   └─────────────────────────────────┘ │"
echo "│                                                                     │"
echo "│  ${CYAN}Channels${NC}                      ${CYAN}Costs${NC}                            │"
echo "│  ┌─────────────────────────┐   ┌─────────────────────────────────┐ │"
echo "│  │ #all        3 messages  │   │ Today:     \$0.12                │ │"
echo "│  │ #eng        0 messages  │   │ This week: \$1.45                │ │"
echo "│  └─────────────────────────┘   └─────────────────────────────────┘ │"
echo "│                                                                     │"
echo "├─────────────────────────────────────────────────────────────────────┤"
echo "│ j/k: navigate │ Enter: select │ q: quit │ ?: help                  │"
echo "└─────────────────────────────────────────────────────────────────────┘"
echo
pause $HERO_PAUSE

# Scene 6: Send work via channel (32-38s)
type_cmd "bc channel send eng 'Build the login feature'"
echo -e "${GREEN}✓${NC} Message sent to #eng"
echo "  Recipients: swift-falcon, bold-hawk"
pause $CMD_PAUSE

# Scene 7: Show costs (38-45s)
type_cmd "bc cost show"
echo
echo -e "${CYAN}Cost Summary${NC}"
echo "┌──────────────┬──────────┬──────────┬──────────┐"
echo "│ AGENT        │ TODAY    │ WEEK     │ TOTAL    │"
echo "├──────────────┼──────────┼──────────┼──────────┤"
echo "│ root         │ \$0.12    │ \$0.89    │ \$2.34    │"
echo "│ swift-falcon │ \$0.08    │ \$0.45    │ \$1.12    │"
echo "│ bold-hawk    │ \$0.05    │ \$0.23    │ \$0.67    │"
echo "├──────────────┼──────────┼──────────┼──────────┤"
echo "│ ${CYAN}TOTAL${NC}        │ \$0.25    │ \$1.57    │ \$4.13    │"
echo "└──────────────┴──────────┴──────────┴──────────┘"
echo
pause $HERO_PAUSE

# Cleanup
rm -rf "$DEMO_DIR"

echo -e "${GREEN}Demo complete!${NC}"
echo
pause 2
