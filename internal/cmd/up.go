package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc agents",
	Long: `Start the bc agent system with the configured roster.

Default roster (configurable in .bc/config.toml [roster]):
  - root (orchestrates work)
  - product-manager (creates epics)
  - manager (assigns tasks)
  - tech-lead-01, tech-lead-02 (technical leadership, code review)
  - engineer-01, engineer-02, engineer-03 (implement tasks)
  - qa-01, qa-02 (test implementations)

Roster configuration in .bc/config.toml:
  [roster]
  engineers = 4   # Number of engineers (0-10)
  tech_leads = 2  # Number of tech-leads (0-10)
  qa = 2          # Number of QA agents (0-10)

CLI flags override config.toml values.

This will:
1. Start all agents in the roster
2. Send role prompts from prompts/ directory
3. Send bootstrap prompts to all agents

Example:
  bc up                      # Start with roster from config.toml
  bc up --engineers 5        # Override engineers count
  bc up --tech-leads 3       # Override tech-leads count
  bc up --qa 3               # Override QA count
  bc up --agent cursor       # Use Cursor AI for all agents`,
	RunE: runUp,
}

var upWorkers int
var upEngineers int
var upTechLeads int
var upQA int
var upAgent string

func init() {
	upCmd.Flags().IntVar(&upWorkers, "workers", 0, "Number of workers (deprecated, use --engineers)")
	upCmd.Flags().IntVar(&upEngineers, "engineers", 3, "Number of engineer agents")
	upCmd.Flags().IntVar(&upTechLeads, "tech-leads", 2, "Number of tech-lead agents")
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

	// Check for existing root state and handle recovery
	rootStore := agent.NewRootStateStore(ws.StateDir())
	recovery, err := rootStore.CheckRecovery(mgr.Tmux())
	if err != nil {
		return fmt.Errorf("failed to check root state: %w", err)
	}

	if recovery.IsRunning {
		// Root is already running
		fmt.Println("Root agent already running!")
		fmt.Printf("  Session: %s\n", recovery.State.Session)
		fmt.Printf("  State: %s\n", recovery.State.State)
		if len(recovery.State.Children) > 0 {
			fmt.Printf("  Children: %s\n", strings.Join(recovery.State.Children, ", "))
		}
		fmt.Println()
		fmt.Println("Use 'bc attach root' to attach or 'bc down' first to restart.")
		return nil
	}

	if recovery.NeedsRecover {
		// Root state exists but session is dead - recover
		fmt.Println("Recovering crashed root agent...")
		fmt.Printf("  Previous session: %s\n", recovery.State.Session)
		if len(recovery.State.Children) > 0 {
			fmt.Printf("  Children to preserve: %s\n", strings.Join(recovery.State.Children, ", "))
		}
		fmt.Println()
	}

	// Determine agent counts from config, with CLI flag overrides
	// Priority: CLI flags > config.toml > hardcoded defaults
	numEngineers := upEngineers
	numTechLeads := upTechLeads
	numQA := upQA

	// Read roster from config if V2Config is available
	if ws.V2Config != nil {
		roster := ws.V2Config.Roster
		// Use config values as base (only if not zero, indicating it was set)
		if roster.Engineers > 0 || roster.Engineers == 0 {
			numEngineers = roster.Engineers
		}
		if roster.TechLeads > 0 || roster.TechLeads == 0 {
			numTechLeads = roster.TechLeads
		}
		if roster.QA > 0 || roster.QA == 0 {
			numQA = roster.QA
		}
	}

	// CLI flags override config values (check if flag was explicitly set)
	if cmd.Flags().Changed("engineers") {
		numEngineers = upEngineers
	}
	if cmd.Flags().Changed("tech-leads") {
		numTechLeads = upTechLeads
	}
	if cmd.Flags().Changed("qa") {
		numQA = upQA
	}

	// Legacy: if --workers is set, use it for engineers (deprecated)
	if upWorkers > 0 {
		numEngineers = upWorkers
	}

	// Event log
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Start root (acts as root agent)
	fmt.Print("Starting root... ")
	coord, err := mgr.SpawnAgent("root", agent.RoleRoot, ws.RootDir)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start root: %w", err)
	}
	fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(coord.Session))

	// Create or update root state
	if recovery.NeedsCreate || recovery.NeedsRecover {
		if recovery.NeedsCreate {
			// Create new root state
			_, createErr := rootStore.Create("root", agent.RoleRoot, "claude")
			if createErr != nil && createErr != agent.ErrRootExists {
				fmt.Printf("  Warning: failed to create root state: %v\n", createErr)
			}
		}
		// Update session in root state
		if updateErr := rootStore.MarkRecovered(coord.Session); updateErr != nil {
			fmt.Printf("  Warning: failed to update root session: %v\n", updateErr)
		}
	}

	_ = log.Append(events.Event{
		Type:  events.AgentSpawned,
		Agent: "root",
	})
	time.Sleep(300 * time.Millisecond)

	// Start product-manager
	fmt.Print("Starting product-manager... ")
	_, err = mgr.SpawnAgent("product-manager", agent.Role("product-manager"), ws.RootDir)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start product-manager: %w", err)
	}
	fmt.Println("✓")
	_ = log.Append(events.Event{Type: events.AgentSpawned, Agent: "product-manager"})
	_ = rootStore.AddChild("product-manager")
	time.Sleep(300 * time.Millisecond)

	// Start manager
	fmt.Print("Starting manager... ")
	_, err = mgr.SpawnAgent("manager", agent.Role("manager"), ws.RootDir)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start manager: %w", err)
	}
	fmt.Println("✓")
	_ = log.Append(events.Event{Type: events.AgentSpawned, Agent: "manager"})
	_ = rootStore.AddChild("manager")
	time.Sleep(300 * time.Millisecond)

	// Start tech-leads
	techLeadNames := make([]string, 0, numTechLeads)
	for i := 1; i <= numTechLeads; i++ {
		name := fmt.Sprintf("tech-lead-%02d", i)
		fmt.Printf("Starting %s... ", name)

		tl, tlErr := mgr.SpawnAgent(name, agent.Role("tech-lead"), ws.RootDir)
		if tlErr != nil {
			fmt.Println("✗")
			fmt.Printf("  Warning: failed to start %s: %v\n", name, tlErr)
			continue
		}
		fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(tl.Session))
		techLeadNames = append(techLeadNames, name)

		_ = log.Append(events.Event{
			Type:  events.AgentSpawned,
			Agent: name,
		})
		_ = rootStore.AddChild(name)

		time.Sleep(300 * time.Millisecond)
	}

	// Start engineers
	engineerNames := make([]string, 0, numEngineers)
	for i := 1; i <= numEngineers; i++ {
		name := fmt.Sprintf("engineer-%02d", i)
		fmt.Printf("Starting %s... ", name)

		eng, err := mgr.SpawnAgent(name, agent.Role("engineer"), ws.RootDir)
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
		_ = rootStore.AddChild(name)

		time.Sleep(300 * time.Millisecond)
	}

	// Start QA agents
	qaNames := make([]string, 0, numQA)
	for i := 1; i <= numQA; i++ {
		name := fmt.Sprintf("qa-%02d", i)
		fmt.Printf("Starting %s... ", name)

		qa, err := mgr.SpawnAgent(name, agent.Role("qa"), ws.RootDir)
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
		_ = rootStore.AddChild(name)

		time.Sleep(300 * time.Millisecond)
	}

	// Create default channels
	allAgents := make([]string, 0, 3+len(techLeadNames)+len(engineerNames)+len(qaNames))
	allAgents = append(allAgents, "root", "product-manager", "manager")
	allAgents = append(allAgents, techLeadNames...)
	allAgents = append(allAgents, engineerNames...)
	allAgents = append(allAgents, qaNames...)
	createDefaultChannels(ws.RootDir, techLeadNames, engineerNames, qaNames, allAgents)

	// Send tech-lead prompts
	techLeadPrompt := loadRolePrompt(ws.RootDir, "tech_lead")
	for _, tlName := range techLeadNames {
		fmt.Printf("Sending bootstrap to %s... ", tlName)
		prompt := techLeadPrompt
		if prompt == "" {
			prompt = fmt.Sprintf("You are a tech lead. Workspace: %s\n\nYour job is to provide technical leadership, review code, and guide engineers. When assigned work:\n1. bc report working \"<task>\"\n2. Review code, provide architectural guidance\n3. bc report done \"<summary>\"\n", ws.RootDir)
		} else {
			prompt += fmt.Sprintf("\n\n---\n\nWorkspace: %s\nYour agent ID: %s\n", ws.RootDir, tlName)
		}
		if tlSendErr := mgr.SendToAgent(tlName, prompt); tlSendErr != nil {
			fmt.Println("✗")
		} else {
			fmt.Println("✓")
		}
	}

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
		if sendErr := mgr.SendToAgent(engName, prompt); sendErr != nil {
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
		if sendErr := mgr.SendToAgent(qaName, prompt); sendErr != nil {
			fmt.Println("✗")
		} else {
			fmt.Println("✓")
		}
	}

	// Build and send bootstrap prompts
	// Coordinator: bootstrap with team info (uses GitHub Issues for work tracking)
	if len(allAgents) > 0 {
		fmt.Print("\nSending bootstrap prompt to root... ")
		prompt := buildBootstrapPrompt(allAgents, ws.RootDir)
		if err := mgr.SendToAgent("root", prompt); err != nil {
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
	teamList := strings.Join(append(append(techLeadNames, engineerNames...), qaNames...), ", ")
	if mgrPrompt == "" {
		mgrPrompt = fmt.Sprintf("You are the manager. Workspace: %s\n\nRun: bc queue && bc status\nTech Leads: %s\nEngineers: %s\nQA: %s\nBreak down epics into tasks and assign to engineers. Tech leads review code. Assign QA to test completed work.\n", ws.RootDir, strings.Join(techLeadNames, ", "), strings.Join(engineerNames, ", "), strings.Join(qaNames, ", "))
	} else {
		// Append dynamic info to the rich prompt
		mgrPrompt += fmt.Sprintf("\n\n---\n\nWorkspace: %s\nTech Leads: %s\nEngineers: %s\nQA: %s\n", ws.RootDir, strings.Join(techLeadNames, ", "), strings.Join(engineerNames, ", "), strings.Join(qaNames, ", "))
	}
	_ = teamList // used in root bootstrap
	if mgrErr := mgr.SendToAgent("manager", mgrPrompt); mgrErr != nil {
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

func buildBootstrapPrompt(agentNames []string, rootDir string) string {
	var b strings.Builder

	b.WriteString("You are the root agent for a bc workspace.\n\n")
	b.WriteString(fmt.Sprintf("Workspace: %s\n", rootDir))
	b.WriteString(fmt.Sprintf("Team: %s\n\n", strings.Join(agentNames, ", ")))

	b.WriteString("=== WORK TRACKING ===\n")
	b.WriteString("Work is tracked via GitHub Issues:\n")
	b.WriteString("  gh issue list --state open     # View open issues\n")
	b.WriteString("  gh issue view <number>         # View issue details\n\n")

	b.WriteString("=== YOUR WORKFLOW ===\n\n")

	b.WriteString("Phase 1 — ASSIGN:\n")
	b.WriteString("  Review GitHub Issues and assign work to team members:\n")
	b.WriteString("    gh issue list --state open\n")
	b.WriteString("    bc send <agent> \"Work on issue #<number>: <instructions>\"\n")
	b.WriteString("  Distribute work evenly across engineers.\n\n")

	b.WriteString("Phase 2 — REVIEW:\n")
	b.WriteString("  After engineers report done, review their branches:\n")
	b.WriteString("    gh pr list --author <agent>\n")
	b.WriteString("    git diff main..<branch>\n")
	b.WriteString("  Verify the implementation matches the issue requirements.\n")
	b.WriteString("  If a branch needs fixes, send feedback via bc send.\n\n")

	b.WriteString("Phase 3 — INTEGRATE:\n")
	b.WriteString("  Merge approved PRs via GitHub:\n")
	b.WriteString("    gh pr merge <number> --squash\n")
	b.WriteString("  Build and test: go build ./... && go test ./...\n")
	b.WriteString("  Report done: bc report done \"all tasks integrated\"\n\n")

	b.WriteString("=== BC COMMANDS ===\n")
	b.WriteString("  bc status          # View agent states\n")
	b.WriteString("  bc agent list      # List all agents\n")
	b.WriteString("  bc send <a> <msg>  # Send message to agent\n")
	b.WriteString("  bc report <state>  # Report your state\n")
	b.WriteString("  bc logs            # View event log")

	return b.String()
}

// createDefaultChannels sets up the default communication channels.
// Channels: #standup (all), #leadership (root, pm, manager, tech-leads),
// #engineering (manager, tech-leads, engineers), #qa (manager, qa), #all (everyone).
// Also creates per-agent channels (#agent-name) for direct messaging.
func createDefaultChannels(rootDir string, techLeadNames, engineerNames, qaNames, allAgents []string) {
	// Use SQLite store for channels (v2)
	store := channel.NewSQLiteStore(rootDir)
	if err := store.Open(); err != nil {
		fmt.Printf("  Warning: failed to open channel store: %v\n", err)
		return
	}
	defer func() { _ = store.Close() }()

	type chanDef struct {
		name        string
		description string
		channelType channel.ChannelType
		members     []string
	}

	// Leadership includes tech-leads
	leadershipMembers := make([]string, 0, 3+len(techLeadNames))
	leadershipMembers = append(leadershipMembers, "root", "product-manager", "manager")
	leadershipMembers = append(leadershipMembers, techLeadNames...)

	// Engineering includes tech-leads and engineers
	engineeringMembers := make([]string, 0, 1+len(techLeadNames)+len(engineerNames))
	engineeringMembers = append(engineeringMembers, "manager")
	engineeringMembers = append(engineeringMembers, techLeadNames...)
	engineeringMembers = append(engineeringMembers, engineerNames...)

	qaMembers := make([]string, 0, 1+len(qaNames))
	qaMembers = append(qaMembers, "manager")
	qaMembers = append(qaMembers, qaNames...)

	// Reviews channel: tech-leads, manager, and engineers can post review requests
	reviewsMembers := make([]string, 0, 1+len(techLeadNames)+len(engineerNames))
	reviewsMembers = append(reviewsMembers, "manager")
	reviewsMembers = append(reviewsMembers, techLeadNames...)
	reviewsMembers = append(reviewsMembers, engineerNames...)

	// Group channels + per-agent channels
	// Preallocate: 6 group channels + 1 per agent
	channels := make([]chanDef, 0, 6+len(allAgents))
	channels = append(channels,
		chanDef{name: "standup", description: "Daily standup channel", channelType: channel.ChannelTypeGroup, members: allAgents},
		chanDef{name: "leadership", description: "Leadership coordination", channelType: channel.ChannelTypeGroup, members: leadershipMembers},
		chanDef{name: "engineering", description: "Engineering team channel", channelType: channel.ChannelTypeGroup, members: engineeringMembers},
		chanDef{name: "qa", description: "QA team channel", channelType: channel.ChannelTypeGroup, members: qaMembers},
		chanDef{name: "reviews", description: "Code review requests and discussions", channelType: channel.ChannelTypeGroup, members: reviewsMembers},
		chanDef{name: "all", description: "Broadcast channel for announcements", channelType: channel.ChannelTypeGroup, members: allAgents},
	)

	// Per-agent channels for direct messaging
	// Each agent gets their own channel with type "direct"
	for _, agentName := range allAgents {
		channels = append(channels, chanDef{
			name:        agentName,
			channelType: channel.ChannelTypeDirect,
			members:     []string{agentName},
			description: fmt.Sprintf("Direct channel for %s", agentName),
		})
	}

	created := 0
	for _, ch := range channels {
		// Check if channel already exists
		existing, _ := store.GetChannel(ch.name)
		if existing != nil {
			// Channel exists, just ensure members are added
			for _, member := range ch.members {
				_ = store.AddMember(ch.name, member)
			}
			continue
		}

		// Create channel with proper type
		if _, createErr := store.CreateChannel(ch.name, ch.channelType, ch.description); createErr != nil {
			fmt.Printf("  Warning: failed to create channel #%s: %v\n", ch.name, createErr)
			continue
		}
		created++

		// Add members
		for _, member := range ch.members {
			_ = store.AddMember(ch.name, member)
		}
	}

	if created > 0 {
		fmt.Printf("Created %d channels (%d group + %d per-agent)\n", created, min(created, 5), max(0, created-5))
	}
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
