package httpapi

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDecodeTerminalCommand(t *testing.T) {
	for _, input := range []string{
		`{"type":"takeover","generation":3}`,
		`{"type":"resize","generation":3,"cols":120,"rows":40}`,
		`{"type":"terminate","generation":3}`,
		`{"type":"cancel_termination","request_id":"request"}`,
	} {
		if _, err := decodeTerminalCommand([]byte(input), 2); err != nil {
			t.Fatalf("valid command %s: %v", input, err)
		}
	}
	for _, input := range []string{
		`{"type":"takeover","generation":3,"generation":4}`,
		`{"type":"terminate","generation":3,"rows":20}`,
		`{"type":"cancel_termination","request_id":""}`,
		`{"type":"unknown"}`,
		`{"type":"resize","generation":-1,"cols":80,"rows":24}`,
		`["terminate"]`,
	} {
		if _, err := decodeTerminalCommand([]byte(input), 2); err == nil {
			t.Fatalf("invalid command accepted: %s", input)
		}
	}
}

func TestDeviceAttributesCommandV3Only(t *testing.T) {
	data, err := os.ReadFile("../../tests/contracts/terminal-runtime.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"primary", "secondary"} {
		input := fixture["device_attributes_"+kind+"_v3"]
		if command, err := decodeTerminalCommand(input, 3); err != nil || command.Kind != kind {
			t.Fatalf("valid device query response: %v", err)
		}
		if _, err := decodeTerminalCommand(input, 2); err == nil {
			t.Fatal("v2 accepted device attributes")
		}
	}
	for _, input := range []string{
		`{"type":"device_attributes"}`,
		`{"type":"device_attributes","kind":null}`,
		`{"type":"device_attributes","kind":"arbitrary"}`,
		`{"type":"device_attributes","kind":"primary","kind":"secondary"}`,
		`{"type":"device_attributes","kind":"primary","payload":"pwd\n"}`,
		`{"type":"device_attributes","kind":"primary","generation":1}`,
		`{"type":"device_attributes","kind":"primary","target":"owner"}`,
	} {
		if _, err := decodeTerminalCommand([]byte(input), 3); err == nil {
			t.Fatalf("invalid device response accepted: %s", input)
		}
	}
}
