package channel

import (
	"testing"
	"time"
)

func TestProcessApprovalMessage(t *testing.T) {
	tests := []struct {
		name    string
		content string
		sender  string
		wantBy  string
		wantPR  int
		wantNil bool
	}{
		{
			name:    "approved PR",
			content: "PR #123 approved",
			sender:  "tech-lead-01",
			wantBy:  "tech-lead-01",
			wantPR:  123,
		},
		{
			name:    "LGTM",
			content: "LGTM PR #456",
			sender:  "tech-lead-02",
			wantBy:  "tech-lead-02",
			wantPR:  456,
		},
		{
			name:    "changes requested - not a merge request",
			content: "PR #789 needs changes",
			sender:  "tech-lead-01",
			wantNil: true,
		},
		{
			name:    "not an approval",
			content: "Hello world",
			sender:  "engineer-01",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ProcessApprovalMessage(tt.content, tt.sender)
			if tt.wantNil {
				if req != nil {
					t.Errorf("expected nil, got %+v", req)
				}
				return
			}
			if req == nil {
				t.Fatal("expected non-nil result")
			}
			if req.PRNumber != tt.wantPR {
				t.Errorf("PRNumber = %d, want %d", req.PRNumber, tt.wantPR)
			}
			if req.ApprovedBy != tt.wantBy {
				t.Errorf("ApprovedBy = %q, want %q", req.ApprovedBy, tt.wantBy)
			}
			if req.TargetBranch != "main" {
				t.Errorf("TargetBranch = %q, want %q", req.TargetBranch, "main")
			}
		})
	}
}

func TestFormatMergeRequest(t *testing.T) {
	req := &MergeRequest{
		PRNumber:     123,
		ApprovedBy:   "tech-lead-01",
		TargetBranch: "main",
		CreatedAt:    time.Now(),
	}

	got := FormatMergeRequest(req)
	want := "@manager PR #123 approved by tech-lead-01 - ready to merge to main"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNewMergeRequestMessage(t *testing.T) {
	req := &MergeRequest{
		PRNumber:     456,
		ApprovedBy:   "tech-lead-02",
		TargetBranch: "main",
		CreatedAt:    time.Now(),
	}

	msg := NewMergeRequestMessage(req, "automation")

	if msg.Type != TypeMerge {
		t.Errorf("Type = %v, want %v", msg.Type, TypeMerge)
	}
	if msg.Sender != "automation" {
		t.Errorf("Sender = %q, want %q", msg.Sender, "automation")
	}
	if msg.Metadata["pr_number"] != "456" {
		t.Errorf("pr_number = %q, want %q", msg.Metadata["pr_number"], "456")
	}
	if msg.Metadata["approved_by"] != "tech-lead-02" {
		t.Errorf("approved_by = %q, want %q", msg.Metadata["approved_by"], "tech-lead-02")
	}
	if msg.Metadata["action"] != "merge_requested" {
		t.Errorf("action = %q, want %q", msg.Metadata["action"], "merge_requested")
	}
}

func TestApprovalHandler_HandleMessage(t *testing.T) {
	var capturedReq *MergeRequest
	handler := &ApprovalHandler{
		OnMergeRequest: func(req *MergeRequest) error {
			capturedReq = req
			return nil
		},
	}

	// Test with approval message
	processed, err := handler.HandleMessage("PR #123 approved", "tech-lead-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Error("expected message to be processed")
	}
	if capturedReq == nil {
		t.Fatal("expected merge request to be captured")
	}
	if capturedReq.PRNumber != 123 {
		t.Errorf("PRNumber = %d, want 123", capturedReq.PRNumber)
	}

	// Test with non-approval message
	capturedReq = nil
	processed, err = handler.HandleMessage("Hello world", "engineer-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Error("expected message NOT to be processed")
	}
	if capturedReq != nil {
		t.Error("expected no merge request")
	}
}

func TestScanHistoryForPendingApprovals(t *testing.T) {
	now := time.Now()
	history := []HistoryEntry{
		{Time: now.Add(-3 * time.Hour), Sender: "engineer-01", Message: "@tech-lead PR #100 ready for review"},
		{Time: now.Add(-2 * time.Hour), Sender: "tech-lead-01", Message: "PR #100 approved"},
		{Time: now.Add(-1 * time.Hour), Sender: "engineer-02", Message: "@tech-lead PR #200 ready for review"},
		{Time: now.Add(-30 * time.Minute), Sender: "tech-lead-01", Message: "PR #200 approved"},
		{Time: now.Add(-15 * time.Minute), Sender: "manager", Message: "PR #100 merged to main"},
	}

	pending := ScanHistoryForPendingApprovals(history)

	// PR #100 was merged, PR #200 is still pending
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending approval, got %d", len(pending))
	}
	if pending[0].PRNumber != 200 {
		t.Errorf("PRNumber = %d, want 200", pending[0].PRNumber)
	}
	if pending[0].Approver != "tech-lead-01" {
		t.Errorf("Approver = %q, want %q", pending[0].Approver, "tech-lead-01")
	}
}

func TestIsPRMerged(t *testing.T) {
	history := []HistoryEntry{
		{Time: time.Now(), Sender: "tech-lead-01", Message: "PR #123 approved"},
		{Time: time.Now(), Sender: "manager", Message: "PR #123 merged to main"},
		{Time: time.Now(), Sender: "tech-lead-01", Message: "PR #456 approved"},
	}

	if !IsPRMerged(history, 123) {
		t.Error("expected PR #123 to be merged")
	}
	if IsPRMerged(history, 456) {
		t.Error("expected PR #456 NOT to be merged")
	}
	if IsPRMerged(history, 789) {
		t.Error("expected PR #789 NOT to be merged")
	}
}

func TestIsPRApproved(t *testing.T) {
	history := []HistoryEntry{
		{Time: time.Now(), Sender: "tech-lead-01", Message: "PR #123 approved"},
		{Time: time.Now(), Sender: "tech-lead-01", Message: "PR #456 needs changes"},
	}

	if !IsPRApproved(history, 123) {
		t.Error("expected PR #123 to be approved")
	}
	if IsPRApproved(history, 456) {
		t.Error("expected PR #456 NOT to be approved (changes requested)")
	}
	if IsPRApproved(history, 789) {
		t.Error("expected PR #789 NOT to be approved")
	}
}

func TestGetPRStatus(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		history []HistoryEntry
		prNum   int
	}{
		{
			name: "merged PR",
			want: "merged",
			history: []HistoryEntry{
				{Message: "PR #123 approved"},
				{Message: "PR #123 merged to main"},
			},
			prNum: 123,
		},
		{
			name: "approved PR",
			want: "approved",
			history: []HistoryEntry{
				{Message: "@tech-lead PR #456 ready for review"},
				{Message: "PR #456 approved"},
			},
			prNum: 456,
		},
		{
			name: "changes requested",
			want: "changes_requested",
			history: []HistoryEntry{
				{Message: "@tech-lead PR #789 ready for review"},
				{Message: "PR #789 needs changes"},
			},
			prNum: 789,
		},
		{
			name: "in review",
			want: "in_review",
			history: []HistoryEntry{
				{Message: "@tech-lead PR #100 ready for review"},
			},
			prNum: 100,
		},
		{
			name:    "unknown PR",
			want:    "unknown",
			history: []HistoryEntry{},
			prNum:   999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetPRStatus(tt.history, tt.prNum)
			if got != tt.want {
				t.Errorf("GetPRStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApprovalToMergeWorkflow(t *testing.T) {
	// Simulate the full workflow
	now := time.Now()

	// 1. Engineer posts review request
	reviewMsg := FormatReviewRequest(123, "tech-lead-01")
	req := ParseReviewRequest(reviewMsg)
	if req == nil || req.PRNumber != 123 {
		t.Fatal("failed to parse review request")
	}

	// 2. Tech lead approves
	approvalMsg := FormatApprovalMessage(123, StatusApproved)
	approval := ParseApprovalMessage(approvalMsg)
	if approval == nil || approval.Status != StatusApproved {
		t.Fatal("failed to parse approval")
	}

	// 3. Automation detects approval and creates merge request
	mergeReq := ProcessApprovalMessage(approvalMsg, "tech-lead-01")
	if mergeReq == nil {
		t.Fatal("failed to create merge request from approval")
	}
	if mergeReq.PRNumber != 123 {
		t.Errorf("merge request PRNumber = %d, want 123", mergeReq.PRNumber)
	}

	// 4. Build history and check status
	history := make([]HistoryEntry, 0, 3)
	history = append(history,
		HistoryEntry{Time: now.Add(-2 * time.Hour), Sender: "engineer-01", Message: reviewMsg},
		HistoryEntry{Time: now.Add(-1 * time.Hour), Sender: "tech-lead-01", Message: approvalMsg},
	)

	status := GetPRStatus(history, 123)
	if status != "approved" {
		t.Errorf("status = %q, want %q", status, "approved")
	}

	// 5. Manager merges
	mergeMsg := FormatMergeNotification(123, "main")
	history = append(history, HistoryEntry{
		Time:    now,
		Sender:  "manager",
		Message: mergeMsg,
	})

	status = GetPRStatus(history, 123)
	if status != "merged" {
		t.Errorf("status = %q, want %q", status, "merged")
	}

	// Verify no pending approvals after merge
	pending := ScanHistoryForPendingApprovals(history)
	if len(pending) != 0 {
		t.Errorf("expected 0 pending approvals, got %d", len(pending))
	}
}
