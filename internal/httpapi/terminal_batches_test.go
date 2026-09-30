package httpapi

import "testing"

func TestBatchProtectionAndStrictPayload(t *testing.T) {
	r, cfg, _ := setup(t)
	cookie, csrf := login(t, r, cfg)
	path := "/api/v1/terminals/termination-batches"
	valid := `{"members":[{"terminal_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","viewer_id":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","generation":1}]}`
	for _, check := range []struct{ origin, csrf string }{{"", csrf}, {"null", csrf}, {"http://evil.test", csrf}, {cfg.Server.PublicOrigin, ""}} {
		if w := request(r, "POST", path, valid, check.origin, cookie, check.csrf); w.Code != 403 {
			t.Fatalf("batch protection: %d", w.Code)
		}
	}
	for _, input := range []string{
		`{}`, `{"members":null}`, `{"members":[]}`, `{"members":[null]}`,
		`{"members":[{"Terminal_ID":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","viewer_id":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","generation":1}]}`,
		`{"members":[{"terminal_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","viewer_id":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","generation":1,"extra":true}]}`,
		`{"members":[{"terminal_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","viewer_id":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","generation":null}]}`,
		`{"members":[{"terminal_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","viewer_id":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","generation":1,"generation":2}]}`,
	} {
		if w := request(r, "POST", path, input, cfg.Server.PublicOrigin, cookie, csrf); w.Code != 400 {
			t.Fatalf("accepted payload %s: %d", input, w.Code)
		}
	}
	if w := request(r, "POST", path, valid, cfg.Server.PublicOrigin, cookie, csrf); w.Code != 409 {
		t.Fatalf("fake viewer accepted: %d", w.Code)
	}
	if w := request(r, "GET", path+"/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "", cfg.Server.PublicOrigin, cookie, csrf); w.Code != 404 {
		t.Fatalf("missing result: %d", w.Code)
	}
}
