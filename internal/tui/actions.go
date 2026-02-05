package tui

// ActionType identifies what action a key handler wants the parent to take.
type ActionType int

const (
	ActionNone ActionType = iota
	ActionBack
	ActionDrillAgent
	ActionDrillIssue
	ActionDrillChannel
	ActionDrillQueue
	ActionAttach
	ActionRefresh
	ActionCreateIssue
)

// Action is returned by sub-screen key handlers to request parent navigation.
type Action struct {
	Type ActionType
	Data any
}

// NoAction is the default no-op action.
var NoAction = Action{Type: ActionNone}
