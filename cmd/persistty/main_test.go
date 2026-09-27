package main

import (
	"bytes"
	"os"
	"testing"
)

func TestCLIRejectsArgvAndNonTTY(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, args := range [][]string{{}, {"unknown"}, {"password", "secret"}, {"password"}, {"serve"}, {"serve", "--config", "/missing", "secret"}} {
		var out bytes.Buffer
		if err := run(args, f, &out, &out); err == nil {
			t.Fatalf("accepted %v", args)
		}
		if bytes.Contains(out.Bytes(), []byte("secret")) {
			t.Fatal("secret echoed")
		}
	}
}
