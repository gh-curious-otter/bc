package channel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestService(t *testing.T) *ChannelService {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}
	store := NewStore(dir)
	return NewChannelService(store)
}

func TestServiceCreate(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     CreateChannelReq
		wantErr bool
	}{
		{
			name:    "valid channel",
			req:     CreateChannelReq{Name: "eng", Description: "Engineering"},
			wantErr: false,
		},
		{
			name:    "without description",
			req:     CreateChannelReq{Name: "ops"},
			wantErr: false,
		},
		{
			name:    "empty name",
			req:     CreateChannelReq{Name: ""},
			wantErr: true,
		},
		{
			name:    "duplicate name",
			req:     CreateChannelReq{Name: "eng"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dto, err := svc.Create(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dto.Name != tt.req.Name {
				t.Errorf("got name %q, want %q", dto.Name, tt.req.Name)
			}
			if dto.Description != tt.req.Description {
				t.Errorf("got desc %q, want %q", dto.Description, tt.req.Description)
			}
			if dto.MemberCount != 0 {
				t.Errorf("got member count %d, want 0", dto.MemberCount)
			}
		})
	}
}

func TestServiceList(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Empty list
	dtos, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dtos) != 0 {
		t.Errorf("got %d channels, want 0", len(dtos))
	}

	// Create some channels
	_, err = svc.Create(ctx, CreateChannelReq{Name: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(ctx, CreateChannelReq{Name: "beta"})
	if err != nil {
		t.Fatal(err)
	}

	dtos, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dtos) != 2 {
		t.Fatalf("got %d channels, want 2", len(dtos))
	}
	if dtos[0].Name != "alpha" {
		t.Errorf("got name %q, want %q", dtos[0].Name, "alpha")
	}
	if dtos[1].Name != "beta" {
		t.Errorf("got name %q, want %q", dtos[1].Name, "beta")
	}
}

func TestServiceGet(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateChannelReq{Name: "eng", Description: "Engineering"})
	if err != nil {
		t.Fatal(err)
	}

	dto, err := svc.Get(ctx, "eng")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.Name != "eng" {
		t.Errorf("got name %q, want %q", dto.Name, "eng")
	}
	if dto.Description != "Engineering" {
		t.Errorf("got desc %q, want %q", dto.Description, "Engineering")
	}

	// Not found
	_, err = svc.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestServiceUpdate(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateChannelReq{Name: "eng"})
	if err != nil {
		t.Fatal(err)
	}

	dto, err := svc.Update(ctx, "eng", UpdateChannelReq{Description: "Updated desc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.Description != "Updated desc" {
		t.Errorf("got desc %q, want %q", dto.Description, "Updated desc")
	}

	// Verify persistence
	dto, err = svc.Get(ctx, "eng")
	if err != nil {
		t.Fatal(err)
	}
	if dto.Description != "Updated desc" {
		t.Errorf("got desc %q, want %q", dto.Description, "Updated desc")
	}

	// Update nonexistent
	_, err = svc.Update(ctx, "nope", UpdateChannelReq{Description: "x"})
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestServiceDelete(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateChannelReq{Name: "eng"})
	if err != nil {
		t.Fatal(err)
	}

	err = svc.Delete(ctx, "eng")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deleted
	_, err = svc.Get(ctx, "eng")
	if err == nil {
		t.Error("expected error after deletion")
	}

	// Delete nonexistent
	if err := svc.Delete(ctx, "nope"); err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestServiceMembers(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateChannelReq{Name: "eng"})
	if err != nil {
		t.Fatal(err)
	}

	// Add members
	err = svc.AddMember(ctx, "eng", "agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = svc.AddMember(ctx, "eng", "agent-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify
	dto, err := svc.Get(ctx, "eng")
	if err != nil {
		t.Fatal(err)
	}
	if dto.MemberCount != 2 {
		t.Errorf("got member count %d, want 2", dto.MemberCount)
	}

	// Remove member
	err = svc.RemoveMember(ctx, "eng", "agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dto, err = svc.Get(ctx, "eng")
	if err != nil {
		t.Fatal(err)
	}
	if dto.MemberCount != 1 {
		t.Errorf("got member count %d, want 1", dto.MemberCount)
	}

	// Add to nonexistent channel
	err = svc.AddMember(ctx, "nope", "agent-1")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}

	// Remove nonexistent member
	err = svc.RemoveMember(ctx, "eng", "ghost")
	if err == nil {
		t.Error("expected error for nonexistent member")
	}
}

func TestServiceSend(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateChannelReq{Name: "eng"})
	if err != nil {
		t.Fatal(err)
	}

	dto, err := svc.Send(ctx, "eng", "agent-1", "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.Channel != "eng" {
		t.Errorf("got channel %q, want %q", dto.Channel, "eng")
	}
	if dto.Sender != "agent-1" {
		t.Errorf("got sender %q, want %q", dto.Sender, "agent-1")
	}
	if dto.Content != "hello world" {
		t.Errorf("got content %q, want %q", dto.Content, "hello world")
	}

	// Send to nonexistent
	_, err = svc.Send(ctx, "nope", "agent-1", "msg")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestServiceHistory(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateChannelReq{Name: "eng"})
	if err != nil {
		t.Fatal(err)
	}

	// Send messages
	for i := 0; i < 10; i++ {
		sender := "agent-1"
		if i%2 == 0 {
			sender = "agent-2"
		}
		_, sendErr := svc.Send(ctx, "eng", sender, fmt.Sprintf("msg-%d", i))
		if sendErr != nil {
			t.Fatal(sendErr)
		}
	}

	tests := []struct {
		name      string
		opts      HistoryOpts
		wantCount int
	}{
		{
			name:      "default limit",
			opts:      HistoryOpts{},
			wantCount: 10,
		},
		{
			name:      "with limit",
			opts:      HistoryOpts{Limit: 3},
			wantCount: 3,
		},
		{
			name:      "filter by agent",
			opts:      HistoryOpts{Agent: "agent-1"},
			wantCount: 5,
		},
		{
			name:      "filter by agent-2",
			opts:      HistoryOpts{Agent: "agent-2"},
			wantCount: 5,
		},
		{
			name:      "with offset",
			opts:      HistoryOpts{Offset: 8, Limit: 50},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dtos, histErr := svc.History(ctx, "eng", tt.opts)
			if histErr != nil {
				t.Fatalf("unexpected error: %v", histErr)
			}
			if len(dtos) != tt.wantCount {
				t.Errorf("got %d messages, want %d", len(dtos), tt.wantCount)
			}
		})
	}

	// History with since filter
	since := time.Now().Add(-1 * time.Second)
	dtos, err := svc.History(ctx, "eng", HistoryOpts{Since: &since})
	if err != nil {
		t.Fatal(err)
	}
	// All messages were just sent, so all should match
	if len(dtos) != 10 {
		t.Errorf("got %d messages with since filter, want 10", len(dtos))
	}

	// Future since should return nothing
	future := time.Now().Add(1 * time.Hour)
	dtos, err = svc.History(ctx, "eng", HistoryOpts{Since: &future})
	if err != nil {
		t.Fatal(err)
	}
	if len(dtos) != 0 {
		t.Errorf("got %d messages with future since, want 0", len(dtos))
	}

	// Nonexistent channel
	_, err = svc.History(ctx, "nope", HistoryOpts{})
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestServiceReact(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, CreateChannelReq{Name: "eng"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Send(ctx, "eng", "agent-1", "test message")
	if err != nil {
		t.Fatal(err)
	}

	// Add reaction
	added, err := svc.React(ctx, "eng", 0, "thumbsup", "agent-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !added {
		t.Error("expected reaction to be added")
	}

	// Toggle off
	added, err = svc.React(ctx, "eng", 0, "thumbsup", "agent-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if added {
		t.Error("expected reaction to be removed on toggle")
	}

	// Invalid message index
	_, err = svc.React(ctx, "eng", 999, "thumbsup", "agent-2")
	if err == nil {
		t.Error("expected error for invalid message index")
	}
}

func TestServiceChannelToDTO(t *testing.T) {
	ch := &Channel{
		Name:        "test",
		Description: "Test channel",
		Members:     []string{"a", "b", "c"},
		History: []HistoryEntry{
			{Time: time.Now(), Sender: "a", Message: "hello"},
			{Time: time.Now(), Sender: "b", Message: "world"},
		},
	}

	dto := channelToDTO(ch)
	if dto.Name != "test" {
		t.Errorf("got name %q, want %q", dto.Name, "test")
	}
	if dto.MemberCount != 3 {
		t.Errorf("got member count %d, want 3", dto.MemberCount)
	}
	if dto.MessageCount != 2 {
		t.Errorf("got message count %d, want 2", dto.MessageCount)
	}
}

func TestServiceChannelToDTONilMembers(t *testing.T) {
	ch := &Channel{
		Name:    "test",
		Members: nil,
	}

	dto := channelToDTO(ch)
	if dto.Members == nil {
		t.Error("expected non-nil members slice")
	}
	if len(dto.Members) != 0 {
		t.Errorf("got %d members, want 0", len(dto.Members))
	}
}
