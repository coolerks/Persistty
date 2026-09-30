package terminal

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestSharedRuntimeFixtureVersionsAndMetadata(t *testing.T) {
	data, err := os.ReadFile("../../tests/contracts/terminal-runtime.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	assertJSON := func(actual []byte, expected json.RawMessage) {
		t.Helper()
		var a, b any
		if err := json.Unmarshal(actual, &a); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(expected, &b); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("contract mismatch: %s != %s", actual, expected)
		}
	}
	h, first, legacy, _ := testHub(t, time.Hour)
	h.terminal.ID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	first.ID = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	first.protocolVersion = 3
	h.controller, h.generation, h.cols, h.rows = first.ID, 3, 80, 24
	h.viewers = map[string]*Viewer{first.ID: first, legacy.ID: legacy}
	var pending struct {
		RequestID string        `json:"request_id"`
		Deadline  time.Time     `json:"deadline"`
		Members   []BatchMember `json:"members"`
	}
	if err := json.Unmarshal(fixture["pending_v3"], &pending); err != nil {
		t.Fatal(err)
	}
	h.pending = &pendingTermination{ID: pending.RequestID, Deadline: pending.Deadline, Members: pending.Members}
	assertJSON(first.Ready(), fixture["ready_v3"])
	var ready map[string]json.RawMessage
	if err := json.Unmarshal(legacy.Ready(), &ready); err != nil {
		t.Fatal(err)
	}
	var oldPending map[string]any
	if err := json.Unmarshal(ready["pending_termination"], &oldPending); err != nil || len(oldPending) != 2 {
		t.Fatal("v2 pending shape changed")
	}
	h.runtime.hubs[h.terminal.ID] = h
	item := h.terminal
	item.DisplayName = "构建"
	h.runtime.UpdateMetadata(item)
	assertJSON((<-first.Frames()).Data, fixture["metadata_v3"])
	select {
	case <-legacy.Frames():
		t.Fatal("metadata leaked to v2")
	default:
	}
	var request struct {
		Members []BatchTarget `json:"members"`
	}
	if err := json.Unmarshal(fixture["batch_request"], &request); err != nil || len(request.Members) != 2 || request.Members[0].Generation != 3 {
		t.Fatal("invalid shared batch request")
	}
	var rename RenameRequest
	if err := json.Unmarshal(fixture["rename"], &rename); err != nil || rename.ExpectedDisplayName != "终端1" || rename.DisplayName != "构建" {
		t.Fatal("invalid shared rename request")
	}
}
