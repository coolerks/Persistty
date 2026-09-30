package httpapi

import (
	"strings"
	"testing"
)

func TestTerminalRenameProtectionAndValidation(t *testing.T) {
	r, cfg, _ := setup(t)
	cookie, csrf := login(t, r, cfg)
	path := "/api/v1/terminals/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, check := range []struct{ origin, csrf string }{{"", csrf}, {"null", csrf}, {"http://evil.test", csrf}, {cfg.Server.PublicOrigin, ""}, {cfg.Server.PublicOrigin, "wrong"}} {
		if w := request(r, "PATCH", path, `{"expected_display_name":"终端","display_name":"构建"}`, check.origin, cookie, check.csrf); w.Code != 403 {
			t.Fatalf("rename protection bypass: %d", w.Code)
		}
	}
	for _, input := range []string{
		`{}`, `{"display_name":"构建"}`, `{"expected_display_name":"终端","display_name":""}`,
		`{"expected_display_name":"终端","display_name":"构建","extra":true}`,
		`{"expected_display_name":"终端","display_name":"构建","display_name":"另一个"}`,
		`{"expected_display_name":"终端","display_name":null}`,
		`{"expected_display_name":"终端","display_name":" leading"}`,
		`{"expected_display_name":"终端","display_name":"` + strings.Repeat("中", 67) + `"}`,
	} {
		if w := request(r, "PATCH", path, input, cfg.Server.PublicOrigin, cookie, csrf); w.Code != 400 {
			t.Fatalf("invalid rename accepted: %d", w.Code)
		}
	}
	if w := request(r, "PATCH", path, `{"expected_display_name":"终端","display_name":"构建"}`, cfg.Server.PublicOrigin, cookie, csrf); w.Code != 404 {
		t.Fatalf("missing terminal recreated: %d", w.Code)
	}
}
