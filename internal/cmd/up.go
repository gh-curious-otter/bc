package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc agents",
	Long: `Start the bc agent system with the default roster.

Default roster:
  - coordinator (orchestrates work)
  - product-manager (creates epics)
  - manager (assigns tasks)
  - engineer-01, engineer-02, engineer-03 (implement tasks)
  - qa-01, qa-02 (test implementations)

This will:
1. Start all agents in the roster
2. Load beads issues into the work queue
3. Send role prompts from prompts/ directory
4. Send bootstrap prompts to all agents

Example:
  bc up                      # Start with default roster (3 engineers, 2 QA)
  bc up --engineers 5        # Start with 5 engineers
  bc up --qa 3               # Start with 3 QA agents
  bc up --agent cursor       # Use Cursor AI for all agents`,
	RunE: runUp,
}

var upWorkers int
var upEngineers int
var upQA int
var upAgent string

func init() {
	upCmd.Flags().IntVar(&upWorkers, "workers", 0, "Number of workers (deprecated, use --engineers)")
	upCmd.Flags().IntVar(&upEngineers, "engineers", 3, "Number of engineer agents")
	upCmd.Flags().IntVar(&upQA, "qa", 2, "Number of QA agents")
	upCmd.Flags().StringVar(&upAgent, "agent", "", "Agent type from config (e.g. claude, cursor, cursor-agent, codex)")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w\nRun 'bc init' first", err)
	}

	fmt.Printf("Starting bc in %s\n\n", ws.RootDir)

	// Create workspace-scoped agent manager
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)

	// Use custom agent command: workspace config > --agent flag > default
	if ws.Config.AgentCommand != "" {
		mgr.SetAgentCommand(ws.Config.AgentCommand)
	} else if upAgent != "" {
		if !mgr.SetAgentByName(upAgent) {
			return fmt.Errorf("unknown agent %q (check config [[agents]] for valid names)", upAgent)
		}
	}

	// Determine agent counts (--workers is deprecated, use --engineers)
	numEngineers := upEngineers
	numQA := upQA
	if upWorkers > 0 {
		// Legacy: if --workers is set, use it for engineers
		numEngineers = upWorkers
	}

	// Event log
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Load beads issues into queue
	q := queue.New(filepath.Join(ws.StateDir(), "queue.json"))
	if err = q.Load(); err != nil {
		return fmt.Errorf("failed to load queue: %w", err)
	}

	issues := beads.ReadyIssues(ws.RootDir)
	if len(issues) == 0 {
		issues, _ = beads.ListIssues(ws.RootDir) //nolint:errcheck // best-effort fallback
	}

	added := 0
	for _, issue := range issues {
		if q.HasBeadsID(issue.ID) {
			continue
		}
		q.Add(issue.Title, issue.Description, issue.ID)
		added++
	}
	if added > 0 {
		if err = q.Save(); err != nil {
			return fmt.Errorf("failed to save queue: %w", err)
		}
		fmt.Printf("Loaded %d items into work queue from beads\n", added)
		_ = log.Append(events.Event{
			Type:    events.QueueLoaded,
			Message: fmt.Sprintf("loaded %d items from beads", added),
			Data:    map[string]any{"added": added},
		})
	}

	// Start coordinator
	fmt.Print("Starting coordinator... ")
	coord, err := mgr.SpawnAgent("coordinator", agent.RoleCoordinator, ws.RootDir)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start coordinator: %w", err)
	}
	fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(coord.Session))

	_ = log.Append(events.Event{
		Type:  events.AgentSpawned,
		Agent: "coordinator",
	})
	time.Sleep(300 * time.Millisecond)

	// Start product-manager
	fmt.Print("Starting product-manager... ")
	_, err = mgr.SpawnAgent("product-manager", agent.RoleProductManager, ws.RootDir)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start product-manager: %w", err)
	}
	fmt.Println("✓")
	_ = log.Append(events.Event{Type: events.AgentSpawned, Agent: "product-manager"})
	time.Sleep(300 * time.Millisecond)

	// Start manager
	fmt.Print("Starting manager... ")
	_, err = mgr.SpawnAgent("manager", agent.RoleManager, ws.RootDir)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start manager: %w", err)
	}
	fmt.Println("✓")
	_ = log.Append(events.Event{Type: events.AgentSpawned, Agent: "manager"})
	time.Sleep(300 * time.Millisecond)

	// Start engineers
	engineerNames := make([]string, 0, numEngineers)
	for i := 1; i <= numEngineers; i++ {
		name := fmt.Sprintf("engineer-%02d", i)
		fmt.Printf("Starting %s... ", name)

		eng, err := mgr.SpawnAgent(name, agent.RoleEngineer, ws.RootDir)
		if err != nil {
			fmt.Println("✗")
			fmt.Printf("  Warning: failed to start %s: %v\n", name, err)
			continue
		}
		fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(eng.Session))
		engineerNames = append(engineerNames, name)

		_ = log.Append(events.Event{
			Type:  events.AgentSpawned,
			Agent: name,
		})

		time.Sleep(300 * time.Millisecond)
	}

	// Start QA agents
	qaNames := make([]string, 0, numQA)
	for i := 1; i <= numQA; i++ {
		name := fmt.Sprintf("qa-%02d", i)
		fmt.Printf("Starting %s... ", name)

		qa, err := mgr.SpawnAgent(name, agent.RoleQA, ws.RootDir)
		if err != nil {
			fmt.Println("✗")
			fmt.Printf("  Warning: failed to start %s: %v\n", name, err)
			continue
		}
		fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(qa.Session))
		qaNames = append(qaNames, name)

		_ = log.Append(events.Event{
			Type:  events.AgentSpawned,
			Agent: name,
		})

		time.Sleep(300 * time.Millisecond)
	}

	// Create default channels
	allAgents := make([]string, 0, 3+len(engineerNames)+len(qaNames))
	allAgents = append(allAgents, "coordinator", "product-manager", "manager")
	allAgents = append(allAgents, engineerNames...)
	allAgents = append(allAgents, qaNames...)
	createDefaultChannels(ws.RootDir, engineerNames, qaNames, allAgents)

	// Send engineer prompts
	engineerPrompt := loadRolePrompt(ws.RootDir, "engineer")
	for _, engName := range engineerNames {
		fmt.Printf("Sending bootstrap to %s... ", engName)
		prompt := engineerPrompt
		if prompt == "" {
			prompt = fmt.Sprintf("You are an engineer. Workspace: %s\n\nWait for assignments from the manager. When assigned work:\n1. bc report working \"<task>\"\n2. Implement the task\n3. bc report done \"<summary>\"\n", ws.RootDir)
		} else {
			prompt += fmt.Sprintf("\n\n---\n\nWorkspace: %s\nYour agent ID: %s\n", ws.RootDir, engName)
		}
		if err := mgr.SendToAgent(engName, prompt); err != nil {
			fmt.Println("✗")
		} else {
			fmt.Println("✓")
		}
	}

	// Send QA prompts
	qaPrompt := loadRolePrompt(ws.RootDir, "qa")
	for _, qaName := range qaNames {
		fmt.Printf("Sending bootstrap to %s... ", qaName)
		prompt := qaPrompt
		if prompt == "" {
			prompt = fmt.Sprintf("You are a QA engineer. Workspace: %s\n\nYour job is to test and validate implementations. When assigned work:\n1. bc report working \"testing <feature>\"\n2. Run tests, review code, check for issues\n3. bc report done \"<test results summary>\"\n", ws.RootDir)
		} else {
			prompt += fmt.Sprintf("\n\n---\n\nWorkspace: %s\nYour agent ID: %s\n", ws.RootDir, qaName)
		}
		if err := mgr.SendToAgent(qaName, prompt); err != nil {
			fmt.Println("✗")
		} else {
			fmt.Println("✓")
		}
	}

	// Build and send bootstrap prompts
	queueItems := q.ListAll()

	// Coordinator: full bootstrap with queue and all agent names (reuse allAgents from above)
	if len(queueItems) > 0 && len(allAgents) > 0 {
		fmt.Print("\nSending bootstrap prompt to coordinator... ")
		prompt := buildBootstrapPrompt(allAgents, queueItems, ws.RootDir)
		if err := mgr.SendToAgent("coordinator", prompt); err != nil {
			fmt.Println("✗")
			fmt.Printf("  Warning: failed to send bootstrap prompt: %v\n", err)
		} else {
			fmt.Println("✓")
		}
	}

	// Product-manager: send rich role prompt from prompts/product_manager.md
	fmt.Print("Sending bootstrap to product-manager... ")
	pmPrompt := loadRolePrompt(ws.RootDir, "product_manager")
	if pmPrompt == "" {
		pmPrompt = fmt.Sprintf("You are the product-manager. Workspace: %s\n\nRun: bc queue && bc status\nThen create or prioritize epics and coordinate with the manager.\n", ws.RootDir)
	}
	if err := mgr.SendToAgent("product-manager", pmPrompt); err != nil {
		fmt.Println("✗")
	} else {
		fmt.Println("✓")
	}

	// Manager: send rich role prompt from prompts/manager.md
	fmt.Print("Sending bootstrap to manager... ")
	mgrPrompt := loadRolePrompt(ws.RootDir, "manager")
	teamList := strings.Join(append(engineerNames, qaNames...), ", ")
	if mgrPrompt == "" {
		mgrPrompt = fmt.Sprintf("You are the manager. Workspace: %s\n\nRun: bc queue && bc status\nEngineers: %s\nQA: %s\nBreak down epics into tasks and assign to engineers. Assign QA to test completed work.\n", ws.RootDir, strings.Join(engineerNames, ", "), strings.Join(qaNames, ", "))
	} else {
		// Append dynamic info to the rich prompt
		mgrPrompt += fmt.Sprintf("\n\n---\n\nWorkspace: %s\nEngineers: %s\nQA: %s\n", ws.RootDir, strings.Join(engineerNames, ", "), strings.Join(qaNames, ", "))
	}
	_ = teamList // used in coordinator bootstrap
	if err := mgr.SendToAgent("manager", mgrPrompt); err != nil {
		fmt.Println("✗")
	} else {
		fmt.Println("✓")
	}

	fmt.Println()
	fmt.Println("All agents started!")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  bc status          # View agent status")
	fmt.Println("  bc attach <agent>  # Attach to an agent's session")
	fmt.Println("  bc down            # Stop all agents")

	return nil
}

// loadRolePrompt reads the prompt file for a role from the prompts/ directory.
// Returns empty string if the file doesn't exist or can't be read.
func loadRolePrompt(rootDir, role string) string {
	promptPath := filepath.Join(rootDir, "prompts", role+".md")
	data, err := os.ReadFile(promptPath) //nolint:gosec // G304: path is constructed from trusted rootDir and role name
	if err != nil {
		return ""
	}
	return string(data)
}

func buildBootstrapPrompt(agentNames []string, items []queue.WorkItem, rootDir string) string {
	var b strings.Builder

	b.WriteString("You are the coordinator agent for a bc workspace.\n\n")
	b.WriteString(fmt.Sprintf("Workspace: %s\n", rootDir))
	b.WriteString(fmt.Sprintf("Team: %s\n\n", strings.Join(agentNames, ", ")))

	b.WriteString("=== WORK QUEUE ===\n")
	for _, item := range items {
		b.WriteString(fmt.Sprintf("\n[%s] %s (beads: %s)\n", item.ID, item.Title, item.BeadsID))
		if item.Description != "" {
			b.WriteString(item.Description)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n=== YOUR WORKFLOW ===\n\n")

	b.WriteString("Phase 1 — ASSIGN:\n")
	b.WriteString("  For each work item, pick a worker and send instructions:\n")
	b.WriteString("    bc queue assign <work-id> <worker>\n")
	b.WriteString("    bc send <worker> \"<detailed instructions>\"\n")
	b.WriteString("  Distribute work evenly across workers.\n\n")

	b.WriteString("Phase 2 — REVIEW:\n")
	b.WriteString("  After workers report done, review their branches (named by bead ID):\n")
	b.WriteString("    git log <bead-id> --oneline  # e.g. git log bc-34b.5\n")
	b.WriteString("    git diff main..<bead-id>\n")
	b.WriteString("  Verify the implementation matches the task description.\n")
	b.WriteString("  If a worker's branch needs fixes, send feedback via bc send.\n\n")

	b.WriteString("Phase 3 — INTEGRATE:\n")
	b.WriteString("  Create an integrate branch and merge all approved worker branches:\n")
	b.WriteString("    git checkout -b integrate main\n")
	b.WriteString("    git merge <branch1> <branch2> ...\n")
	b.WriteString("  Build and test: go build ./... && go test ./...\n")
	b.WriteString("  Report done: bc report done \"all tasks integrated\"\n\n")

	b.WriteString("=== BC COMMANDS ===\n")
	b.WriteString("  bc status          # View agent states\n")
	b.WriteString("  bc queue           # View work queue\n")
	b.WriteString("  bc queue assign    # Assign work to agent\n")
	b.WriteString("  bc send <a> <msg>  # Send message to agent\n")
	b.WriteString("  bc report <state>  # Report your state\n")
	b.WriteString("  bc logs            # View event log")

	return b.String()
}

// createDefaultChannels sets up the default communication channels.
// Channels: #standup (all), #leadership (coordinator, pm, manager),
// #engineering (manager, engineers), #qa (manager, qa), #all (everyone).
func createDefaultChannels(rootDir string, engineerNames, qaNames, allAgents []string) {
	store := channel.NewStore(rootDir)
	if err := store.Load(); err != nil {
		fmt.Printf("  Warning: failed to load channels: %v\n", err)
	}

	type chanDef struct {
		name    string
		members []string
	}

	leadershipMembers := []string{"coordinator", "product-manager", "manager"}

	engineeringMembers := make([]string, 0, 1+len(engineerNames))
	engineeringMembers = append(engineeringMembers, "manager")
	engineeringMembers = append(engineeringMembers, engineerNames...)

	qaMembers := make([]string, 0, 1+len(qaNames))
	qaMembers = append(qaMembers, "manager")
	qaMembers = append(qaMembers, qaNames...)

	channels := []chanDef{
		{"standup", allAgents},
		{"leadership", leadershipMembers},
		{"engineering", engineeringMembers},
		{"qa", qaMembers},
		{"all", allAgents},
	}

	created := 0
	for _, ch := range channels {
		// Create channel if it doesn't already exist
		if _, exists := store.Get(ch.name); !exists {
			if _, err := store.Create(ch.name); err != nil {
				fmt.Printf("  Warning: failed to create channel #%s: %v\n", ch.name, err)
				continue
			}
		}
		// Add members (skip if already present)
		for _, member := range ch.members {
			_ = store.AddMember(ch.name, member)
		}
		created++
	}

	if created > 0 {
		if err := store.Save(); err != nil {
			fmt.Printf("  Warning: failed to save channels: %v\n", err)
			return
		}
		fmt.Printf("Created %d default channels\n", created)
	}
}
