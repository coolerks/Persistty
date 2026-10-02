package elevation

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestFramesRejectDuplicateUnknownMalformedAndOversize(t *testing.T) {
	for _, raw := range []string{`{"phase":"ready","phase":"commit"}`, `{"phase":"ready","extra":1}`, `{"phase":"ready","Phase":"commit"}`, `{"phase":`, "{\"phase\":\"\xff\"}", `{"phase":"ready"} {}`} {
		var frame bytes.Buffer
		if err := WriteBytes(&frame, []byte(raw), 64<<10); err != nil {
			t.Fatal(err)
		}
		var v struct {
			Phase string `json:"phase"`
		}
		if ReadFrame(&frame, &v) == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	var h [4]byte
	binary.BigEndian.PutUint32(h[:], MaxContent+1)
	if _, err := ReadBytes(bytes.NewReader(h[:]), MaxContent); err == nil {
		t.Fatal("oversize accepted")
	}
	var buf bytes.Buffer
	body := []byte("\uFEFFa\r\nb\nc")
	if WriteBytes(&buf, body, MaxContent) != nil {
		t.Fatal("write")
	}
	got, err := ReadBytes(&buf, MaxContent)
	if err != nil || !bytes.Equal(got, body) {
		t.Fatal("changed bytes", err)
	}
	for _, password := range [][]byte{nil, []byte("a\nb"), []byte("a\r"), []byte{0}, bytes.Repeat([]byte{'x'}, 1025)} {
		if ValidPassword(password) {
			t.Fatal("invalid password accepted")
		}
	}
}
func TestPolicyDenyInfrastructureAndBroadPaths(t *testing.T) {
	base := Policy{Schema: 1, CallerUID: 1000, CallerGID: 1000, LedgerPath: "/var/lib/persistty-elevation", ProtectedPaths: []string{"/srv/private-web"}, Targets: []Target{{"example", "/srv/allowed/file.txt"}}}
	raw, _ := json.Marshal(base)
	if _, err := ParsePolicy(raw); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{PolicyPath, HelperPath, BrokerPath, "/etc/persistty-elevation/other", "/etc/sudoers.d/example", "/etc/pam.d/sudo", "/usr/bin/sudo", "/etc/systemd/system/example.service", "/var/lib/persistty-elevation/nonce", "/srv/private-web/config", "/srv/allowed/../escape", "relative"} {
		p := base
		p.Targets = []Target{{"example", path}}
		raw, _ = json.Marshal(p)
		if _, err := ParsePolicy(raw); err == nil {
			t.Fatal("allowed", path)
		}
	}
	if _, err := ParsePolicy([]byte(strings.Replace(string(raw), `"schema":1`, `"schema":1,"schema":1`, 1))); err == nil {
		t.Fatal("duplicate policy key")
	}
}

func TestSharedHTTPFixture(t *testing.T) {
	data, err := os.ReadFile("../../tests/contracts/elevation.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Prepared struct {
			Data      Prepared `json:"data"`
			RequestID string   `json:"request_id"`
		} `json:"prepared"`
		Results []struct {
			Data      Result `json:"data"`
			RequestID string `json:"request_id"`
		} `json:"results"`
	}
	if strictJSON(data, &fixture) != nil {
		t.Fatal("invalid shared fixture")
	}
	if fixture.Prepared.Data.State != "prepared" || !noncePattern.MatchString(fixture.Prepared.Data.ID) {
		t.Fatal("prepared fixture")
	}
	for _, row := range fixture.Results {
		if !ResultValid(row.Data, fixture.Prepared.Data.ID) {
			t.Fatal(row)
		}
	}
}
