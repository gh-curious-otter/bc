## bc channel

Manage communication channels

### Synopsis

Manage channels for broadcasting messages to groups of agents.

Channels are named groups of agent members. Messages sent to a channel are
delivered to all member tmux sessions.

Examples:
  bc channel list                      # List all channels
  bc channel create workers            # Create a channel named "workers"
  bc channel show workers              # Show channel details
  bc channel add workers worker-01     # Add member to channel
  bc channel add workers --agent w-01  # Add member via --agent flag
  bc channel send workers "run tests"  # Send to all members
  bc channel history workers --last 20 # Show last 20 messages
  bc channel react workers 5 👍        # React to message
  bc channel edit workers --desc "..."  # Edit channel description
  bc channel remove workers worker-01  # Remove a member
  bc channel delete workers            # Delete the channel
  bc channel status                    # Overview of all channels

Agent Commands (require BC_AGENT_ID):
  bc channel join workers              # Join a channel (current agent)
  bc channel leave workers             # Leave a channel (current agent)

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
  bc status           View agents and their channels

### Options

```
  -h, --help   help for channel
```

### Options inherited from parent commands

```
      --json      Output in JSON format
  -v, --verbose   Enable verbose output
```

### SEE ALSO

* [bc](bc.md)	 - A simpler, more controllable agent orchestrator
* [bc channel add](bc_channel_add.md)	 - Add members to a channel
* [bc channel create](bc_channel_create.md)	 - Create a new channel
* [bc channel delete](bc_channel_delete.md)	 - Delete a channel
* [bc channel desc](bc_channel_desc.md)	 - Set channel description
* [bc channel edit](bc_channel_edit.md)	 - Edit channel description/settings
* [bc channel history](bc_channel_history.md)	 - Show channel message history
* [bc channel join](bc_channel_join.md)	 - Join a channel (for agents)
* [bc channel leave](bc_channel_leave.md)	 - Leave a channel (for agents)
* [bc channel list](bc_channel_list.md)	 - List all channels
* [bc channel react](bc_channel_react.md)	 - React to a channel message
* [bc channel remove](bc_channel_remove.md)	 - Remove a member from a channel
* [bc channel send](bc_channel_send.md)	 - Send a message to all channel members
* [bc channel show](bc_channel_show.md)	 - Show channel details
* [bc channel status](bc_channel_status.md)	 - Show channel overview with activity details

