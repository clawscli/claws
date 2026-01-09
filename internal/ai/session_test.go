package ai

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSessionManager(t *testing.T) {
	t.Run("default max sessions", func(t *testing.T) {
		sm := NewSessionManager(0, false)
		if sm.maxSessions != DefaultMaxSessions {
			t.Errorf("expected maxSessions %d, got %d", DefaultMaxSessions, sm.maxSessions)
		}
	})

	t.Run("custom max sessions", func(t *testing.T) {
		sm := NewSessionManager(50, false)
		if sm.maxSessions != 50 {
			t.Errorf("expected maxSessions %d, got %d", 50, sm.maxSessions)
		}
	})

	t.Run("save disabled", func(t *testing.T) {
		sm := NewSessionManager(10, false)
		if sm.saveEnabled {
			t.Error("expected saveEnabled to be false")
		}
	})

	t.Run("save enabled", func(t *testing.T) {
		sm := NewSessionManager(10, true)
		if !sm.saveEnabled {
			t.Error("expected saveEnabled to be true")
		}
	})
}

func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	time.Sleep(time.Millisecond)
	id2 := generateSessionID()

	if id1 == "" {
		t.Error("expected non-empty session ID")
	}
	if id1 == id2 {
		t.Error("expected unique session IDs")
	}

	// Check format: YYYY-MM-DD-xxxxxx
	if len(id1) < 10 {
		t.Errorf("session ID too short: %q", id1)
	}
}

func TestSessionManagerNewSession(t *testing.T) {
	sm := NewSessionManager(10, false) // save disabled

	ctx := &Context{
		Service:      "ec2",
		ResourceType: "instances",
	}

	session, err := sm.NewSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.ID == "" {
		t.Error("expected non-empty session ID")
	}
	if session.Context == nil {
		t.Error("expected context")
	}
	if session.Context.Service != "ec2" {
		t.Errorf("expected service %q, got %q", "ec2", session.Context.Service)
	}
	if len(session.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(session.Messages))
	}
	if session.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
	if session.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestSessionManagerAddMessage(t *testing.T) {
	sm := NewSessionManager(10, false)

	session, err := sm.NewSession(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg := NewUserMessage("test message")
	err = sm.AddMessage(session, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(session.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(session.Messages))
	}
	if session.Messages[0].Content[0].Text != "test message" {
		t.Errorf("expected message %q, got %q", "test message", session.Messages[0].Content[0].Text)
	}
}

func TestSessionManagerWithPersistence(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "claws-session-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Setenv("HOME", tmpDir)

	sm := NewSessionManager(10, true) // save enabled

	session, err := sm.NewSession(&Context{Service: "lambda"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Add a message
	err = sm.AddMessage(session, NewUserMessage("hello"))
	if err != nil {
		t.Fatalf("failed to add message: %v", err)
	}

	// Verify file was created
	sessionsDir := filepath.Join(tmpDir, ".config", "claws", "chat", "sessions")
	sessionFile := filepath.Join(sessionsDir, session.ID+".json")
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Error("expected session file to be created")
	}

	// Load session
	loaded, err := sm.LoadSession(session.ID)
	if err != nil {
		t.Fatalf("failed to load session: %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("expected ID %q, got %q", session.ID, loaded.ID)
	}
	if len(loaded.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(loaded.Messages))
	}
	if loaded.Context.Service != "lambda" {
		t.Errorf("expected service %q, got %q", "lambda", loaded.Context.Service)
	}
}

func TestContext(t *testing.T) {
	t.Run("single mode", func(t *testing.T) {
		ctx := &Context{
			Mode:           ContextModeSingle,
			Service:        "ec2",
			ResourceType:   "instances",
			ResourceID:     "i-12345",
			ResourceName:   "my-instance",
			ResourceRegion: "us-east-1",
		}
		if ctx.Mode != ContextModeSingle {
			t.Errorf("expected mode %q, got %q", ContextModeSingle, ctx.Mode)
		}
		if ctx.ResourceID != "i-12345" {
			t.Errorf("expected resource ID %q, got %q", "i-12345", ctx.ResourceID)
		}
	})

	t.Run("list mode", func(t *testing.T) {
		ctx := &Context{
			Mode:          ContextModeList,
			Service:       "lambda",
			ResourceType:  "functions",
			ResourceCount: 25,
			FilterText:    "prod",
		}
		if ctx.Mode != ContextModeList {
			t.Errorf("expected mode %q, got %q", ContextModeList, ctx.Mode)
		}
		if ctx.ResourceCount != 25 {
			t.Errorf("expected count %d, got %d", 25, ctx.ResourceCount)
		}
	})

	t.Run("diff mode", func(t *testing.T) {
		ctx := &Context{
			Mode:         ContextModeDiff,
			Service:      "rds",
			ResourceType: "instances",
			DiffLeft: &ResourceRef{
				ID:     "db-1",
				Name:   "prod-db",
				Region: "us-east-1",
			},
			DiffRight: &ResourceRef{
				ID:     "db-2",
				Name:   "staging-db",
				Region: "us-west-2",
			},
		}
		if ctx.Mode != ContextModeDiff {
			t.Errorf("expected mode %q, got %q", ContextModeDiff, ctx.Mode)
		}
		if ctx.DiffLeft.ID != "db-1" {
			t.Errorf("expected left ID %q, got %q", "db-1", ctx.DiffLeft.ID)
		}
		if ctx.DiffRight.Region != "us-west-2" {
			t.Errorf("expected right region %q, got %q", "us-west-2", ctx.DiffRight.Region)
		}
	})
}

func TestResourceRef(t *testing.T) {
	ref := ResourceRef{
		ID:      "i-12345",
		Name:    "my-instance",
		Region:  "us-east-1",
		Profile: "prod",
		Cluster: "my-cluster",
	}

	if ref.ID != "i-12345" {
		t.Errorf("expected ID %q, got %q", "i-12345", ref.ID)
	}
	if ref.Cluster != "my-cluster" {
		t.Errorf("expected cluster %q, got %q", "my-cluster", ref.Cluster)
	}
}

func TestSessionListEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "claws-session-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Setenv("HOME", tmpDir)

	sm := NewSessionManager(10, true)

	sessions, err := sm.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestSessionPruning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "claws-session-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Setenv("HOME", tmpDir)

	sm := NewSessionManager(3, true) // Max 3 sessions

	// Create 5 sessions with messages (files only saved on AddMessage)
	for i := 0; i < 5; i++ {
		sess, err := sm.NewSession(nil)
		if err != nil {
			t.Fatalf("failed to create session %d: %v", i, err)
		}
		// AddMessage triggers file save and pruning (on first message)
		if err := sm.AddMessage(sess, NewUserMessage("test")); err != nil {
			t.Fatalf("failed to add message to session %d: %v", i, err)
		}
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	// List sessions - should be pruned to 3
	sessions, err := sm.ListSessions()
	if err != nil {
		t.Fatalf("failed to list sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions after pruning, got %d", len(sessions))
	}
}

func TestShouldPrune(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpDir)

	sm := NewSessionManager(5, true)

	t.Run("no sessions", func(t *testing.T) {
		should, err := sm.shouldPrune()
		if err != nil {
			t.Fatalf("shouldPrune failed: %v", err)
		}
		if should {
			t.Error("empty dir should not need pruning")
		}
	})

	t.Run("under limit", func(t *testing.T) {
		// Create 4 sessions (below limit of 5)
		for i := 0; i < 4; i++ {
			sess, err := sm.NewSession(nil)
			if err != nil {
				t.Fatalf("NewSession failed: %v", err)
			}
			if err := sm.SaveMessages(sess); err != nil {
				t.Fatalf("SaveMessages failed: %v", err)
			}
		}

		should, err := sm.shouldPrune()
		if err != nil {
			t.Fatalf("shouldPrune failed: %v", err)
		}
		if should {
			t.Error("under limit should not need pruning")
		}
	})

	t.Run("at limit", func(t *testing.T) {
		// Add one more session (total 5, at limit)
		sess, err := sm.NewSession(nil)
		if err != nil {
			t.Fatalf("NewSession failed: %v", err)
		}
		if err := sm.SaveMessages(sess); err != nil {
			t.Fatalf("SaveMessages failed: %v", err)
		}

		should, err := sm.shouldPrune()
		if err != nil {
			t.Fatalf("shouldPrune failed: %v", err)
		}
		if should {
			t.Error("at limit should not need pruning")
		}
	})

	t.Run("over limit", func(t *testing.T) {
		// Add one more session (total 6, over limit)
		sess, err := sm.NewSession(nil)
		if err != nil {
			t.Fatalf("NewSession failed: %v", err)
		}
		if err := sm.SaveMessages(sess); err != nil {
			t.Fatalf("SaveMessages failed: %v", err)
		}

		should, err := sm.shouldPrune()
		if err != nil {
			t.Fatalf("shouldPrune failed: %v", err)
		}
		if !should {
			t.Error("over limit should need pruning")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		tmpDir2 := t.TempDir()
		os.Setenv("HOME", tmpDir2)
		defer os.Setenv("HOME", tmpDir)

		sm2 := NewSessionManager(5, true)
		should, err := sm2.shouldPrune()
		if err != nil {
			t.Fatalf("shouldPrune failed: %v", err)
		}
		if should {
			t.Error("non-existent dir should not need pruning")
		}
	})
}
