package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc agents",
	Long: `Start the bc agent system.

This will:
1. Start the coordinator agent
2. Start the product-manager and manager agents
3. Start worker agents (based on max-workers or --workers)
4. Load beads issues into the work queue
5. Send bootstrap prompts to coordinator, product-manager, and manager

Example:
  bc up                    # Start with default settings
  bc up --workers 5        # Start with 5 workers
  bc up --agent cursor    # Use Cursor AI for all agents`,
	RunE: runUp,
}

var upWorkers int
var upAgent string

func init() {
	upCmd.Flags().IntVar(&upWorkers, "workers", 0, "Number of workers (0 = use config default)")
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

	// Determine worker count
	numWorkers := ws.Config.MaxWorkers
	if upWorkers > 0 {
		numWorkers = upWorkers
	}

	// Event log
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Load beads issues into queue
	q := queue.New(filepath.Join(ws.StateDir(), "queue.json"))
	q.Load()

	issues := beads.ReadyIssues(ws.RootDir)
	if len(issues) == 0 {
		issues = beads.ListIssues(ws.RootDir)
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
		q.Save()
		fmt.Printf("Loaded %d items into work queue from beads\n", added)
		log.Append(events.Event{
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

	log.Append(events.Event{
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
	log.Append(events.Event{Type: events.AgentSpawned, Agent: "product-manager"})
	time.Sleep(300 * time.Millisecond)

	// Start manager
	fmt.Print("Starting manager... ")
	_, err = mgr.SpawnAgent("manager", agent.RoleManager, ws.RootDir)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start manager: %w", err)
	}
	fmt.Println("✓")
	log.Append(events.Event{Type: events.AgentSpawned, Agent: "manager"})
	time.Sleep(300 * time.Millisecond)

	// Start workers
	workerNames := make([]string, 0, numWorkers)
	for i := 1; i <= numWorkers; i++ {
		name := fmt.Sprintf("worker-%02d", i)
		fmt.Printf("Starting %s... ", name)

		worker, err := mgr.SpawnAgent(name, agent.RoleWorker, ws.RootDir)
		if err != nil {
			fmt.Println("✗")
			fmt.Printf("  Warning: failed to start %s: %v\n", name, err)
			continue
		}
		fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(worker.Session))
		workerNames = append(workerNames, name)

		log.Append(events.Event{
			Type:  events.AgentSpawned,
			Agent: name,
		})

		// Small delay between spawns
		time.Sleep(300 * time.Millisecond)
	}

	// Build and send bootstrap prompts
	queueItems := q.ListAll()

	// Coordinator: full bootstrap with queue and all agent names (product-manager, manager, workers)
	allAgents := append([]string{"product-manager", "manager"}, workerNames...)
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

	// Product-manager: brief bootstrap so it starts working
	fmt.Print("Sending bootstrap to product-manager... ")
	pmPrompt := fmt.Sprintf("You are the product-manager. Workspace: %s\n\nRun: bc queue && bc status\nThen create or prioritize epics and coordinate with the manager.\n", ws.RootDir)
	if err := mgr.SendToAgent("product-manager", pmPrompt); err != nil {
		fmt.Println("✗")
	} else {
		fmt.Println("✓")
	}

	// Manager: brief bootstrap so it starts working
	fmt.Print("Sending bootstrap to manager... ")
	mgrPrompt := fmt.Sprintf("You are the manager. Workspace: %s\n\nRun: bc queue && bc status\nWorkers: %s\nBreak down epics into tasks and assign to workers. Coordinate with product-manager for priorities.\n", ws.RootDir, strings.Join(workerNames, ", "))
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
	b.WriteString("  bc logs            # View event log\n")

	return b.String()
}
