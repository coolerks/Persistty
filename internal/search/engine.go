package search

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"persistty/internal/toolrunner"
	"sort"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type rgText struct {
	Text  *string `json:"text"`
	Bytes *string `json:"bytes"`
}
type rgSub struct {
	Start       int     `json:"start"`
	End         int     `json:"end"`
	Match       rgText  `json:"match"`
	Replacement *rgText `json:"replacement"`
}
type rgMessage struct {
	Type string `json:"type"`
	Data struct {
		Lines      rgText  `json:"lines"`
		LineNumber int     `json:"line_number"`
		Offset     int     `json:"absolute_offset"`
		Submatches []rgSub `json:"submatches"`
	} `json:"data"`
}
type expanded struct {
	Match       Match
	Replacement string
}

func rgEngine(ctx context.Context, r *toolrunner.Runner, dir string, q Query, content string, replacement *string) ([]expanded, error) {
	args := []string{"--no-config", "--encoding", "none", "--json", "--engine", "default", "--crlf", "--max-count", "5001"}
	if !q.Regex {
		args = append(args, "--fixed-strings")
	}
	if !q.CaseSensitive {
		args = append(args, "--ignore-case")
	}
	if q.WholeWord {
		args = append(args, "--word-regexp")
	}
	if replacement != nil {
		value := *replacement
		if !q.Regex {
			value = strings.ReplaceAll(value, "$", "$$")
		}
		args = append(args, "--replace", value)
	}
	args = append(args, "-e", q.Pattern, "--", "-")
	input := []byte(content)
	for i := range input {
		if input[i] == '\r' && (i+1 == len(input) || input[i+1] != '\n') {
			input[i] = '\n'
		}
	}
	out, code, err := r.Run(ctx, "rg", dir, input, args...)
	if err != nil {
		return nil, err
	}
	if code == 2 {
		return nil, ErrPattern
	}
	if code != 0 && code != 1 {
		return nil, toolrunner.ErrUnavailable
	}
	lineStarts := []int{0}
	for i := 0; i < len(content); i++ {
		if content[i] == '\r' {
			if i+1 < len(content) && content[i+1] == '\n' {
				i++
			}
			lineStarts = append(lineStarts, i+1)
		} else if content[i] == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}
	result := []expanded{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var message rgMessage
		err := dec.Decode(&message)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, toolrunner.ErrUnavailable
		}
		if message.Type != "match" {
			continue
		}
		d := message.Data
		if d.Lines.Text == nil || d.LineNumber < 1 || d.Offset < 0 {
			return nil, toolrunner.ErrUnavailable
		}
		line := *d.Lines.Text
		if d.Offset+len(line) > len(content) || string(input[d.Offset:d.Offset+len(line)]) != line {
			return nil, toolrunner.ErrUnavailable
		}
		for _, sub := range d.Submatches {
			if sub.Start < 0 || sub.End < sub.Start || sub.End > len(line) || !utf8.ValidString(line[:sub.Start]) || !utf8.ValidString(line[:sub.End]) {
				return nil, toolrunner.ErrUnavailable
			}
			start := d.Offset + sub.Start
			lineIndex := sort.Search(len(lineStarts), func(i int) bool { return lineStarts[i] > start }) - 1
			prefix := content[lineStarts[lineIndex]:start]
			if lineIndex == 0 {
				prefix = strings.TrimPrefix(prefix, "\uFEFF")
			}
			lineNumber := lineIndex + 1
			column := 1 + len(utf16.Encode([]rune(prefix)))
			m := Match{Line: lineNumber, Column: column, EndColumn: column + len(utf16.Encode([]rune(line[sub.Start:sub.End]))), Preview: strings.TrimRight(line, "\r\n"), Start: d.Offset + sub.Start, End: d.Offset + sub.End}
			if len(m.Preview) > 2048 {
				m.Preview = string([]rune(m.Preview)[:min(len([]rune(m.Preview)), 512)])
			}
			value := ""
			if replacement != nil {
				if sub.Replacement == nil || sub.Replacement.Text == nil {
					return nil, toolrunner.ErrUnavailable
				}
				value = *sub.Replacement.Text
			}
			result = append(result, expanded{m, value})
			if len(result) > 5000 {
				return result, nil
			}
		}
	}
	return result, nil
}
