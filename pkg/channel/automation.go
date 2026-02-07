// Package channel - automation.go provides approval-to-merge workflow automation.
package channel

import (
	"fmt"
	"time"
)

// ApprovalEvent represents a detected PR approval that needs merge action.
type ApprovalEvent struct {
	// DetectedAt is when the approval was detected
	DetectedAt time.Time `json:"detected_at"`

	// Approver is who approved the PR
	Approver string `json:"approver"`

	// Comment is any approval comment
	Comment string `json:"comment,omitempty"`

	// PRNumber is the pull request number
	PRNumber int `json:"pr_number"`
}

// MergeRequest represents a request for manager to merge an approved PR.
type MergeRequest struct {
	// CreatedAt is when the merge request was created
	CreatedAt time.Time `json:"created_at"`

	// ApprovedBy is who approved the PR
	ApprovedBy string `json:"approved_by"`

	// TargetBranch is the merge target (usually "main")
	TargetBranch string `json:"target_branch"`

	// PRNumber is the pull request number
	PRNumber int `json:"pr_number"`
}

// ProcessApprovalMessage checks if a message is an approval and creates a merge request.
// Returns nil if the message is not an approval.
func ProcessApprovalMessage(content, sender string) *MergeRequest {
	approval := ParseApprovalMessage(content)
	if approval == nil {
		return nil
	}

	if approval.Status != StatusApproved {
		return nil
	}

	return &MergeRequest{
		PRNumber:     approval.PRNumber,
		ApprovedBy:   sender,
		TargetBranch: "main",
		CreatedAt:    time.Now(),
	}
}

// FormatMergeRequest creates a message notifying manager of merge-ready PR.
func FormatMergeRequest(req *MergeRequest) string {
	return fmt.Sprintf("@manager PR #%d approved by %s - ready to merge to %s",
		req.PRNumber, req.ApprovedBy, req.TargetBranch)
}

// NewMergeRequestMessage creates a TypedMessage for a merge request.
func NewMergeRequestMessage(req *MergeRequest, sender string) *TypedMessage {
	content := FormatMergeRequest(req)
	msg := NewTypedMessage(content, TypeMerge, sender)
	msg.WithMetadata("pr_number", fmt.Sprintf("%d", req.PRNumber))
	msg.WithMetadata("approved_by", req.ApprovedBy)
	msg.WithMetadata("target_branch", req.TargetBranch)
	msg.WithMetadata("action", "merge_requested")
	return msg
}

// ApprovalHandler processes channel messages and triggers merge workflow.
type ApprovalHandler struct {
	// OnMergeRequest is called when a merge request should be created
	OnMergeRequest func(req *MergeRequest) error
}

// HandleMessage processes a message and triggers merge workflow if it's an approval.
// Returns true if the message was an approval and was processed.
func (h *ApprovalHandler) HandleMessage(content, sender string) (bool, error) {
	req := ProcessApprovalMessage(content, sender)
	if req == nil {
		return false, nil
	}

	if h.OnMergeRequest != nil {
		if err := h.OnMergeRequest(req); err != nil {
			return true, fmt.Errorf("failed to process merge request: %w", err)
		}
	}

	return true, nil
}

// WatchChannelForApprovals creates a handler that watches for approvals in a channel.
// This is a factory function for setting up the approval workflow.
func WatchChannelForApprovals(store *Store, channelName string) (*ApprovalHandler, error) {
	_, exists := store.Get(channelName)
	if !exists {
		return nil, fmt.Errorf("channel %q not found", channelName)
	}

	handler := &ApprovalHandler{
		OnMergeRequest: func(req *MergeRequest) error {
			// Add merge request to channel history
			msg := FormatMergeRequest(req)
			return store.AddHistory(channelName, "automation", msg)
		},
	}

	return handler, nil
}

// ScanHistoryForPendingApprovals scans channel history for approvals that may need merge.
// Returns PRs that have been approved but not yet merged.
func ScanHistoryForPendingApprovals(history []HistoryEntry) []ApprovalEvent {
	approvals := make(map[int]*ApprovalEvent)
	merges := make(map[int]bool)

	for _, entry := range history {
		// Check for approvals
		if approval := ParseApprovalMessage(entry.Message); approval != nil {
			if approval.Status == StatusApproved {
				approvals[approval.PRNumber] = &ApprovalEvent{
					PRNumber:   approval.PRNumber,
					Approver:   entry.Sender,
					Comment:    approval.Comment,
					DetectedAt: entry.Time,
				}
			}
		}

		// Check for merges
		if merge := ParseMergeNotification(entry.Message); merge != nil {
			merges[merge.PRNumber] = true
		}
	}

	// Return approvals that haven't been merged
	var pending []ApprovalEvent
	for prNum, approval := range approvals {
		if !merges[prNum] {
			pending = append(pending, *approval)
		}
	}

	return pending
}

// IsPRMerged checks if a PR has been merged based on channel history.
func IsPRMerged(history []HistoryEntry, prNumber int) bool {
	for _, entry := range history {
		if merge := ParseMergeNotification(entry.Message); merge != nil {
			if merge.PRNumber == prNumber {
				return true
			}
		}
	}
	return false
}

// IsPRApproved checks if a PR has been approved based on channel history.
func IsPRApproved(history []HistoryEntry, prNumber int) bool {
	for _, entry := range history {
		if approval := ParseApprovalMessage(entry.Message); approval != nil {
			if approval.PRNumber == prNumber && approval.Status == StatusApproved {
				return true
			}
		}
	}
	return false
}

// GetPRStatus returns the current workflow status of a PR based on channel history.
func GetPRStatus(history []HistoryEntry, prNumber int) string {
	var hasReview, hasApproval, hasMerge, hasChangesRequested bool

	for _, entry := range history {
		if IsReviewRequest(entry.Message) {
			if req := ParseReviewRequest(entry.Message); req != nil && req.PRNumber == prNumber {
				hasReview = true
			}
		}

		if approval := ParseApprovalMessage(entry.Message); approval != nil {
			if approval.PRNumber == prNumber {
				switch approval.Status {
				case StatusApproved:
					hasApproval = true
				case StatusChangesRequested:
					hasChangesRequested = true
				}
			}
		}

		if merge := ParseMergeNotification(entry.Message); merge != nil {
			if merge.PRNumber == prNumber {
				hasMerge = true
			}
		}
	}

	if hasMerge {
		return "merged"
	}
	if hasChangesRequested {
		return "changes_requested"
	}
	if hasApproval {
		return "approved"
	}
	if hasReview {
		return "in_review"
	}
	return "unknown"
}
