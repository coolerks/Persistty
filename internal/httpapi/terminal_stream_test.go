package httpapi

import "testing"

func TestDecodeTerminalCommand(t *testing.T) {
	for _, input := range []string{
		`{"type":"takeover","generation":3}`,
		`{"type":"resize","generation":3,"cols":120,"rows":40}`,
		`{"type":"terminate","generation":3}`,
		`{"type":"cancel_termination","request_id":"request"}`,
	} {
		if _, err := decodeTerminalCommand([]byte(input)); err != nil {
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
		if _, err := decodeTerminalCommand([]byte(input)); err == nil {
			t.Fatalf("invalid command accepted: %s", input)
		}
	}
}
