package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestTerminalMetadata(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "private", "db.sqlite"), "testhash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	created := Terminal{
		ID: "terminal-id", TmuxSessionName: "persistty_terminal_id",
		DisplayName: "构建", WorkingDirectory: "/tmp/project",
	}
	if err := store.InsertTerminal(ctx, created); err != nil {
		t.Fatal(err)
	}
	byID, err := store.Terminal(ctx, created.ID)
	if err != nil || byID.TmuxSessionName != created.TmuxSessionName || byID.State != "unavailable" {
		t.Fatalf("terminal by ID = %+v, %v", byID, err)
	}
	bySession, err := store.TerminalBySession(ctx, created.TmuxSessionName)
	if err != nil || bySession.ID != created.ID {
		t.Fatalf("terminal by session = %+v, %v", bySession, err)
	}
	listed, err := store.Terminals(ctx)
	if err != nil || len(listed) != 1 || listed[0].TmuxSessionName != created.TmuxSessionName {
		t.Fatalf("terminals = %+v, %v", listed, err)
	}
	if err := store.InsertTerminal(ctx, created); err == nil {
		t.Fatal("duplicate terminal was accepted")
	}
	if _, err := store.Terminal(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing terminal: %v", err)
	}
	if _, err := store.TerminalBySession(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing session: %v", err)
	}
}
