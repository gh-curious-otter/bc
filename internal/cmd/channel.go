package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/spf13/cobra"
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
	rootCmd.AddCommand(channelCmd)
}

func loadChannelStore(rootDir string) (*channel.Store, error) {
	store := channel.NewStore(rootDir)
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

	// Send to all members
	sent := 0
	failed := 0
	for _, member := range members {
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

		if err := mgr.SendToAgent(member, fmt.Sprintf("[#%s] %s", channelName, message)); err != nil {
			fmt.Printf("  %s: failed - %v\n", member, err)
			failed++
			continue
		}
		fmt.Printf("  %s: sent\n", member)
		sent++
	}

	fmt.Printf("\nSent to %d/%d members of channel %q\n", sent, len(members), channelName)
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
	for _, entry := range history {
		if entry.Sender != "" {
			fmt.Printf("[%s] %s: %s\n", entry.Time.Format("15:04:05"), entry.Sender, entry.Message)
		} else {
			fmt.Printf("[%s] %s\n", entry.Time.Format("15:04:05"), entry.Message)
		}
	}

	return nil
}
