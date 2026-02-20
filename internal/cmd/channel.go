package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage communication channels",
	Long: `Manage channels for broadcasting messages to groups of agents.

Channels are named groups of agent members. Messages sent to a channel are
delivered to all member tmux sessions.

Examples:
  bc channel list                      # List all channels
  bc channel create workers            # Create a channel named "workers"
  bc channel add workers worker-01     # Add member to channel
  bc channel send workers "run tests"  # Send to all members
  bc channel remove workers worker-01  # Remove a member
  bc channel delete workers            # Delete the channel
  bc channel join workers              # Join a channel (current agent)
  bc channel leave workers             # Leave a channel (current agent)
  bc channel history workers           # Show channel message history

Default Channels:
  #eng       Engineering team (all engineer agents)
  #pr        Pull request reviews and notifications
  #standup   Daily standup updates
  #leads     Tech leads and managers

Message Format:
  Messages are delivered as system reminders to agent sessions.
  Use @agent-name to mention specific agents in messages.

See Also:
  bc agent send       Send message to single agent
  bc agent broadcast  Send to all agents
  bc status           View agents and their channels`,
}

var channelCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new channel",
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelCreate,
}

var channelAddCmd = &cobra.Command{
	Use:   "add <channel> <member> [member...]",
	Short: "Add members to a channel",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runChannelAdd,
}

var channelRemoveCmd = &cobra.Command{
	Use:   "remove <channel> <member>",
	Short: "Remove a member from a channel",
	Args:  cobra.ExactArgs(2),
	RunE:  runChannelRemove,
}

var channelSendCmd = &cobra.Command{
	Use:   "send <channel> <message>",
	Short: "Send a message to all channel members",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runChannelSend,
}

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all channels",
	RunE:  runChannelList,
}

var channelDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a channel",
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelDelete,
}

var channelJoinCmd = &cobra.Command{
	Use:   "join <channel>",
	Short: "Join a channel (for agents)",
	Long:  `Add yourself to a channel. This command must be run from within an agent session.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelJoin,
}

var channelLeaveCmd = &cobra.Command{
	Use:   "leave <channel>",
	Short: "Leave a channel (for agents)",
	Long:  `Remove yourself from a channel. This command must be run from within an agent session.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelLeave,
}

var channelHistoryCmd = &cobra.Command{
	Use:   "history <channel>",
	Short: "Show channel message history",
	Long: `Display the history of messages sent to a channel.

Examples:
  bc channel history eng                # Last 50 messages (default)
  bc channel history eng --limit 10     # Last 10 messages
  bc channel history eng --since 1h     # Messages from last hour
  bc channel history eng --limit 20 --offset 20  # Page 2 of 20`,
	Args: cobra.ExactArgs(1),
	RunE: runChannelHistory,
}

var channelReactCmd = &cobra.Command{
	Use:   "react <channel> <message-index> <emoji>",
	Short: "React to a channel message",
	Long: `Add an emoji reaction to a message in a channel.

The message-index is shown in 'bc channel history' output.
Use common emoji like 👍, 👎, ❤️, 🎉, 👀, 🚀 or any emoji.

Examples:
  bc channel react engineering 5 👍
  bc channel react general 0 🎉`,
	Args: cobra.ExactArgs(3),
	RunE: runChannelReact,
}

var channelShowCmd = &cobra.Command{
	Use:   "show <channel>",
	Short: "Show channel details",
	Long: `Display detailed information about a channel including members,
description, and message history statistics.

Examples:
  bc channel show engineering    # Show engineering channel details
  bc channel show standup --json # Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runChannelShow,
}

var channelDescCmd = &cobra.Command{
	Use:   "desc <channel> <description>",
	Short: "Set channel description",
	Long:  `Set or update the description for a channel.`,
	Args:  cobra.MinimumNArgs(2),
	RunE:  runChannelDesc,
}

var channelCreateDesc string

// Channel history flags
var (
	channelHistoryLimit  int
	channelHistoryOffset int
	channelHistorySince  string
)

func init() {
	channelCreateCmd.Flags().StringVar(&channelCreateDesc, "desc", "", "Channel description")
	channelHistoryCmd.Flags().IntVar(&channelHistoryLimit, "limit", 50, "Maximum number of messages to show")
	channelHistoryCmd.Flags().IntVar(&channelHistoryOffset, "offset", 0, "Number of messages to skip")
	channelHistoryCmd.Flags().StringVar(&channelHistorySince, "since", "", "Show messages since duration (e.g., 1h, 30m)")
	channelCmd.AddCommand(channelCreateCmd)
	channelCmd.AddCommand(channelAddCmd)
	channelCmd.AddCommand(channelRemoveCmd)
	channelCmd.AddCommand(channelSendCmd)
	channelCmd.AddCommand(channelListCmd)
	channelCmd.AddCommand(channelDeleteCmd)
	channelCmd.AddCommand(channelJoinCmd)
	channelCmd.AddCommand(channelLeaveCmd)
	channelCmd.AddCommand(channelHistoryCmd)
	channelCmd.AddCommand(channelReactCmd)
	channelCmd.AddCommand(channelShowCmd)
	channelCmd.AddCommand(channelDescCmd)
	rootCmd.AddCommand(channelCmd)
}

func loadChannelStore(rootDir string) (*channel.Store, error) {
	store, err := channel.OpenStore(rootDir)
	if err != nil {
		return nil, err
	}
	if err := store.Load(); err != nil {
		return nil, err
	}
	return store, nil
}

func runChannelList(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channels := store.List()

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		// Build enhanced channel list with member counts and descriptions for TUI
		type ChannelSummary struct {
			Name        string   `json:"name"`
			Description string   `json:"description,omitempty"`
			Members     []string `json:"members"`
			MemberCount int      `json:"member_count"`
		}
		summaries := make([]ChannelSummary, 0, len(channels))
		for _, ch := range channels {
			desc, _ := store.GetDescription(ch.Name)
			summaries = append(summaries, ChannelSummary{
				Name:        ch.Name,
				Members:     ch.Members,
				MemberCount: len(ch.Members),
				Description: desc,
			})
		}
		response := struct {
			Channels []ChannelSummary `json:"channels"`
		}{Channels: summaries}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(response)
	}

	if len(channels) == 0 {
		fmt.Println("No channels defined")
		fmt.Println()
		fmt.Println("Run 'bc channel create <name>' to create a channel")
		fmt.Println("Or run 'bc up' to create default channels")
		return nil
	}

	// Table header
	fmt.Printf("%-20s %-8s %s\n", "CHANNEL", "MEMBERS", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 70))

	for _, ch := range channels {
		memberCount := fmt.Sprintf("(%d)", len(ch.Members))
		desc := ""
		if d, _ := store.GetDescription(ch.Name); d != "" {
			desc = truncateMessage(d, 30)
		}
		fmt.Printf("%-20s %-8s %s\n", ch.Name, memberCount, desc)
	}

	return nil
}

func runChannelCreate(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("channel name cannot be empty")
	}
	// Validate channel name to prevent log injection and special character issues
	if !validIdentifier(name) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", name)
	}
	if _, err := store.Create(name); err != nil {
		return err
	}

	// Set description if provided
	if channelCreateDesc != "" {
		if err := store.SetDescription(name, channelCreateDesc); err != nil {
			return fmt.Errorf("failed to set description: %w", err)
		}
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save channels: %w", err)
	}

	if channelCreateDesc != "" {
		fmt.Printf("Created channel %q with description: %s\n", name, channelCreateDesc)
	} else {
		fmt.Printf("Created channel %q\n", name)
	}
	return nil
}

func runChannelAdd(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
	// Validate channel name to prevent log injection
	if !validIdentifier(channelName) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", channelName)
	}
	members := args[1:]

	added := 0
	for _, member := range members {
		if err := store.AddMember(channelName, member); err != nil {
			fmt.Printf("  Warning: %v\n", err)
			continue
		}
		added++
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save channels: %w", err)
	}

	fmt.Printf("Added %d member(s) to channel %q\n", added, channelName)
	return nil
}

func runChannelRemove(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
	// Validate channel name to prevent log injection
	if !validIdentifier(channelName) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", channelName)
	}
	member := args[1]

	if err := store.RemoveMember(channelName, member); err != nil {
		return err
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save channels: %w", err)
	}

	fmt.Printf("Removed %q from channel %q\n", member, channelName)
	return nil
}

func runChannelSend(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
	// Validate channel name to prevent log injection
	if !validIdentifier(channelName) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", channelName)
	}
	message := strings.Join(args[1:], " ")

	members, err := store.GetMembers(channelName)
	if err != nil {
		return err
	}

	if len(members) == 0 {
		fmt.Printf("Channel %q has no members\n", channelName)
		return nil
	}

	// Create workspace-scoped agent manager
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if err := mgr.LoadState(); err != nil {
		fmt.Printf("Warning: failed to load agent state: %v\n", err)
	}

	// Add to channel history
	sender := os.Getenv("BC_AGENT_ID")
	if sender == "" {
		sender = "cli"
	}
	if err := store.AddHistory(channelName, sender, message); err != nil {
		fmt.Printf("Warning: failed to record history: %v\n", err)
	}
	if err := store.Save(); err != nil {
		fmt.Printf("Warning: failed to save history: %v\n", err)
	}

	// Send to all members except the sender
	sent := 0
	failed := 0
	skipped := 0
	fmt.Printf("Sending to %d member(s):\n", len(members))
	for _, member := range members {
		// Skip sending to the sender to avoid infinite loop
		if member == sender {
			skipped++
			continue
		}

		a := mgr.GetAgent(member)
		if a == nil {
			fmt.Printf("  ❌ %s: agent not found\n", member)
			failed++
			continue
		}
		if a.State == agent.StateStopped {
			fmt.Printf("  ⏸  %s: agent stopped\n", member)
			failed++
			continue
		}

		if err := mgr.SendToAgent(member, fmt.Sprintf("[#%s] %s: %s", channelName, sender, message)); err != nil {
			fmt.Printf("  ❌ %s: unable to deliver message\n", member)
			failed++
			continue
		}
		fmt.Printf("  ✅ %s: sent\n", member)
		sent++
	}

	totalTargets := len(members) - skipped
	if totalTargets == 0 {
		fmt.Printf("\nMessage recorded to channel (no other members to deliver to)\n")
	} else {
		fmt.Printf("\nResult: %d/%d members received message\n", sent, totalTargets)
		if failed > 0 {
			fmt.Printf("Warning: %d delivery failed\n", failed)
		}
	}
	return nil
}

func runChannelDelete(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	name := args[0]
	// Validate channel name to prevent log injection
	if !validIdentifier(name) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", name)
	}
	if err := store.Delete(name); err != nil {
		return err
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save channels: %w", err)
	}

	fmt.Printf("Deleted channel %q\n", name)
	return nil
}

func runChannelJoin(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	agentID := os.Getenv("BC_AGENT_ID")
	if agentID == "" {
		return errorAgentNotRunning(fmt.Sprintf("bc channel join %s", args[0]))
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
	// Validate channel name to prevent log injection
	if !validIdentifier(channelName) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", channelName)
	}
	if err := store.AddMember(channelName, agentID); err != nil {
		return err
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save channels: %w", err)
	}

	fmt.Printf("Joined channel %q\n", channelName)
	return nil
}

func runChannelLeave(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	agentID := os.Getenv("BC_AGENT_ID")
	if agentID == "" {
		return errorAgentNotRunning(fmt.Sprintf("bc channel leave %s", args[0]))
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
	// Validate channel name to prevent log injection
	if !validIdentifier(channelName) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", channelName)
	}
	if err := store.RemoveMember(channelName, agentID); err != nil {
		return err
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save channels: %w", err)
	}

	fmt.Printf("Left channel %q\n", channelName)
	return nil
}

func runChannelHistory(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
	// Validate channel name to prevent log injection
	if !validIdentifier(channelName) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", channelName)
	}
	history, err := store.GetHistory(channelName)
	if err != nil {
		return err
	}

	// Filter by --since if provided
	if channelHistorySince != "" {
		cutoff, parseErr := parseSinceDuration(channelHistorySince)
		if parseErr != nil {
			return parseErr
		}
		filtered := history[:0]
		for _, entry := range history {
			if !entry.Time.Before(cutoff) {
				filtered = append(filtered, entry)
			}
		}
		history = filtered
	}

	// Apply --offset and --limit
	if channelHistoryOffset > 0 {
		if channelHistoryOffset >= len(history) {
			history = nil
		} else {
			history = history[channelHistoryOffset:]
		}
	}
	if channelHistoryLimit > 0 && len(history) > channelHistoryLimit {
		history = history[len(history)-channelHistoryLimit:]
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		// Wrap in object for TUI compatibility
		response := struct {
			Channel  string                 `json:"channel"`
			Messages []channel.HistoryEntry `json:"messages"`
		}{Channel: channelName, Messages: history}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(response)
	}

	if len(history) == 0 {
		fmt.Printf("No message history for channel %q\n", channelName)
		return nil
	}

	fmt.Printf("Message history for #%s:\n", channelName)
	fmt.Println(strings.Repeat("-", 60))
	for i, entry := range history {
		if entry.Sender != "" {
			fmt.Printf("[%d] [%s] %s: %s\n", i, entry.Time.Format("15:04:05"), entry.Sender, entry.Message)
		} else {
			fmt.Printf("[%d] [%s] %s\n", i, entry.Time.Format("15:04:05"), entry.Message)
		}
		// Show reactions if any
		if len(entry.Reactions) > 0 {
			var reactionStrs []string
			for emoji, users := range entry.Reactions {
				reactionStrs = append(reactionStrs, fmt.Sprintf("%s %d", emoji, len(users)))
			}
			fmt.Printf("    Reactions: %s\n", strings.Join(reactionStrs, " "))
		}
	}

	return nil
}

func runChannelReact(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
	// Validate channel name to prevent log injection
	if !validIdentifier(channelName) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", channelName)
	}
	messageIndex, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid message index %q: %w", args[1], err)
	}
	emoji := args[2]

	// Get user identity
	user := os.Getenv("BC_AGENT_ID")
	if user == "" {
		user = "cli"
	}

	added, err := store.ToggleReaction(channelName, messageIndex, emoji, user)
	if err != nil {
		return err
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save reactions: %w", err)
	}

	if added {
		fmt.Printf("Added %s reaction to message %d in #%s\n", emoji, messageIndex, channelName)
	} else {
		fmt.Printf("Removed %s reaction from message %d in #%s\n", emoji, messageIndex, channelName)
	}
	return nil
}

// ChannelInfo represents detailed channel information for JSON output.
type ChannelInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	Members      []string `json:"members"`
	MemberCount  int      `json:"member_count"`
	HistoryCount int      `json:"history_count"`
}

func runChannelShow(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
	// Validate channel name to prevent log injection
	if !validIdentifier(channelName) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", channelName)
	}

	// Get channel
	ch, exists := store.Get(channelName)
	if !exists {
		return fmt.Errorf("channel %q not found", channelName)
	}

	// Get members
	members, err := store.GetMembers(channelName)
	if err != nil {
		return fmt.Errorf("failed to get members: %w", err)
	}

	// Get history for count
	history, err := store.GetHistory(channelName)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	// Get description
	description, _ := store.GetDescription(channelName)

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	if jsonOutput {
		info := ChannelInfo{
			Name:         ch.Name,
			Description:  description,
			Members:      members,
			MemberCount:  len(members),
			HistoryCount: len(history),
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	}

	// Text output
	fmt.Printf("Channel: #%s\n", ch.Name)
	fmt.Println(strings.Repeat("-", 40))

	if description != "" {
		fmt.Printf("Description: %s\n", description)
	}

	fmt.Printf("Members (%d):\n", len(members))
	if len(members) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, m := range members {
			fmt.Printf("  • %s\n", m)
		}
	}

	fmt.Printf("\nMessage History: %d messages\n", len(history))

	if len(history) > 0 {
		fmt.Println("\nRecent Messages (last 5):")
		start := 0
		if len(history) > 5 {
			start = len(history) - 5
		}
		for i := start; i < len(history); i++ {
			entry := history[i]
			msg := strings.ReplaceAll(entry.Message, "\n", " ")
			fmt.Printf("  [%s] %s: %s\n", entry.Time.Format("15:04"), entry.Sender, truncateMessage(msg, 50))
		}
	}

	return nil
}

func runChannelDesc(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := strings.TrimSpace(args[0])
	if channelName == "" {
		return fmt.Errorf("channel name cannot be empty")
	}
	// Validate channel name to prevent log injection
	if !validIdentifier(channelName) {
		return fmt.Errorf("channel name %q contains invalid characters (use letters, numbers, dash, underscore)", channelName)
	}

	// Join description from remaining arguments
	description := strings.TrimSpace(strings.Join(args[1:], " "))
	if description == "" {
		return fmt.Errorf("description cannot be empty")
	}

	if err := store.SetDescription(channelName, description); err != nil {
		return fmt.Errorf("failed to set description: %w", err)
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save channels: %w", err)
	}

	fmt.Printf("Updated description for channel %q: %s\n", channelName, description)
	return nil
}
