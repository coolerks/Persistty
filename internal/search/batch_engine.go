package search

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"persistty/internal/toolrunner"
	"strconv"
)

// One rg process consumes a bounded batch of already verified text snapshots.
// Synthetic ASCII names decouple CLI paths from source names (including LF).
func rgBatch(ctx context.Context, runner *toolrunner.Runner, parent string, q Query, batch []File) ([][]expanded, []error, error) {
	dir, err := os.MkdirTemp(parent, "match-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(dir)
	args := append(rgArguments(q, nil), "--no-ignore", "--hidden", "--text", "--")
	for i, file := range batch {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		name := strconv.Itoa(i)
		if err := os.WriteFile(filepath.Join(dir, name), normalizedInput(file.Content), 0600); err != nil {
			return nil, nil, err
		}
		args = append(args, name)
	}
	out, code, err := runner.Run(ctx, "rg", dir, nil, args...)
	if err != nil {
		return nil, nil, err
	}
	if code == 2 {
		return nil, nil, ErrPattern
	}
	if code != 0 && code != 1 {
		return nil, nil, toolrunner.ErrUnavailable
	}
	grouped := make([]bytes.Buffer, len(batch))
	decoder := json.NewDecoder(bytes.NewReader(out))
	for {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err == io.EOF {
			break
		} else if err != nil {
			return nil, nil, toolrunner.ErrUnavailable
		}
		var message rgMessage
		if err := json.Unmarshal(raw, &message); err != nil {
			return nil, nil, toolrunner.ErrUnavailable
		}
		if message.Type != "match" {
			continue
		}
		if message.Data.Path.Text == nil {
			return nil, nil, toolrunner.ErrUnavailable
		}
		name := *message.Data.Path.Text
		index, err := strconv.Atoi(name)
		if err != nil || index < 0 || index >= len(batch) || strconv.Itoa(index) != name {
			return nil, nil, toolrunner.ErrUnavailable
		}
		grouped[index].Write(raw)
		grouped[index].WriteByte('\n')
	}
	values, failures := make([][]expanded, len(batch)), make([]error, len(batch))
	for i, file := range batch {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		values[i], failures[i] = parseRG(grouped[i].Bytes(), file.Content, nil)
	}
	return values, failures, nil
}
