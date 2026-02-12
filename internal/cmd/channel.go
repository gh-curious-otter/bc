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
  bc channel                         # list all channels
  bc channel list                    # list all channels
  bc channel create workers          # create a channel named "workers"
  bc channel add workers worker-01   # add member to channel
  bc channel send workers "run tests"  # send to all members
  bc channel remove workers worker-01  # remove a member
  bc channel delete workers          # delete the channel
  bc channel join workers            # join a channel (current agent)
  bc channel leave workers           # leave a channel (current agent)
  bc channel history workers         # show channel message history`,
	RunE: runChannelList,
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
	Short: "Join a channel (uses BC_AGENT_ID)",
	Long:  `Add yourself to a channel. Uses the BC_AGENT_ID environment variable to identify the current agent.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelJoin,
}

var channelLeaveCmd = &cobra.Command{
	Use:   "leave <channel>",
	Short: "Leave a channel (uses BC_AGENT_ID)",
	Long:  `Remove yourself from a channel. Uses the BC_AGENT_ID environment variable to identify the current agent.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelLeave,
}

var channelHistoryCmd = &cobra.Command{
	Use:   "history <channel>",
	Short: "Show channel message history",
	Long:  `Display the history of messages sent to a channel.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelHistory,
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

func init() {
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
		return fmt.Errorf("not in a bc workspace: %w", err)
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
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(channels)
	}

	if len(channels) == 0 {
		fmt.Println("No channels defined")
		fmt.Println()
		fmt.Println("Run 'bc channel create <name>' to create a channel")
		fmt.Println("Or run 'bc up' to create default channels")
		return nil
	}

	// Table header
	fmt.Printf("%-20s %s\n", "CHANNEL", "MEMBERS")
	fmt.Println(strings.Repeat("-", 60))

	for _, ch := range channels {
		members := "-"
		if len(ch.Members) > 0 {
			members = strings.Join(ch.Members, ", ")
		}
		fmt.Printf("%-20s %s\n", ch.Name, members)
	}

	return nil
}

func runChannelCreate(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
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
	if _, err := store.Create(name); err != nil {
		return err
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save channels: %w", err)
	}

	fmt.Printf("Created channel %q\n", name)
	return nil
}

func runChannelAdd(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
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
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
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
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
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
	for _, member := range members {
		// Skip sending to the sender to avoid infinite loop
		if member == sender {
			skipped++
			continue
		}

		a := mgr.GetAgent(member)
		if a == nil {
			fmt.Printf("  %s: agent not found\n", member)
			failed++
			continue
		}
		if a.State == agent.StateStopped {
			fmt.Printf("  %s: agent stopped\n", member)
			failed++
			continue
		}

		if err := mgr.SendToAgent(member, fmt.Sprintf("[#%s] %s: %s", channelName, sender, message)); err != nil {
			fmt.Printf("  %s: failed - %v\n", member, err)
			failed++
			continue
		}
		fmt.Printf("  %s: sent\n", member)
		sent++
	}

	totalTargets := len(members) - skipped
	fmt.Printf("\nSent to %d/%d members of channel %q\n", sent, totalTargets, channelName)
	if skipped > 0 {
		fmt.Printf("  (%d skipped - sender)\n", skipped)
	}
	if failed > 0 {
		fmt.Printf("  (%d failed)\n", failed)
	}
	return nil
}

func runChannelDelete(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	name := args[0]
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
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	agentID := os.Getenv("BC_AGENT_ID")
	if agentID == "" {
		return fmt.Errorf("BC_AGENT_ID not set - this command is for agents to use")
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
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
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	agentID := os.Getenv("BC_AGENT_ID")
	if agentID == "" {
		return fmt.Errorf("BC_AGENT_ID not set - this command is for agents to use")
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
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
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
	history, err := store.GetHistory(channelName)
	if err != nil {
		return err
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(history)
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
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]
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
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	store, err := loadChannelStore(ws.RootDir)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	channelName := args[0]

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
		enc := json.NewEncoder(os.Stdout)
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
