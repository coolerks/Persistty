package gitview

import (
	"bytes"
	"net/url"
	"persistty/internal/files"
	"regexp"
	"strconv"
	"strings"
)

var githubPart = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// Extract only a canonical public web URL, never remote credentials or options.
func githubRepository(remote string) string {
	if strings.HasPrefix(strings.ToLower(remote), "git@github.com:") {
		remote = "ssh://git@github.com/" + remote[len("git@github.com:"):]
	}
	u, err := url.Parse(remote)
	if err != nil || !strings.EqualFold(u.Hostname(), "github.com") || u.RawQuery != "" || u.Fragment != "" {
		return ""
	}
	if u.Scheme != "https" && u.Scheme != "ssh" || u.Port() != "" && !(u.Scheme == "https" && u.Port() == "443") && !(u.Scheme == "ssh" && u.Port() == "22") {
		return ""
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimSuffix(u.Path, "/"), ".git"), "/")
	if len(parts) != 3 || parts[0] != "" {
		return ""
	}
	for _, p := range parts[1:] {
		if !githubPart.MatchString(p) || p == "." || p == ".." {
			return ""
		}
	}
	return "https://github.com/" + parts[1] + "/" + parts[2]
}

// --raw --numstat -z emits raw records first, then counts. Renames have two
// NUL-delimited paths in both sections; tabs/newlines in names remain literal.
func parseFileStats(b []byte) ([]FileStat, error) {
	out := []FileStat{}
	if len(b) == 0 {
		return out, nil
	}
	if b[len(b)-1] != 0 {
		return nil, ErrUnavailable
	}
	parts := bytes.Split(b[:len(b)-1], []byte{0})
	byPath := map[string]int{}
	i := 0
	for i < len(parts) && bytes.HasPrefix(parts[i], []byte(":")) {
		fields := strings.Fields(string(parts[i]))
		i++
		if len(fields) != 5 || !objectID(fields[2]) || !objectID(fields[3]) || len(fields[4]) == 0 || i >= len(parts) {
			return nil, ErrUnavailable
		}
		status := fields[4][:1]
		if !strings.Contains("AMDTRC", status) {
			return nil, ErrUnavailable
		}
		p, old := string(parts[i]), ""
		i++
		if status == "R" || status == "C" {
			if i >= len(parts) {
				return nil, ErrUnavailable
			}
			old, p = p, string(parts[i])
			i++
		}
		if !files.ValidRelative(p, false) || old != "" && !files.ValidRelative(old, false) {
			return nil, ErrUnavailable
		}
		if _, exists := byPath[p]; exists {
			return nil, ErrUnavailable
		}
		byPath[p] = len(out)
		out = append(out, FileStat{Path: p, OldPath: old, Status: status})
		if len(out) > 5000 {
			return nil, files.ErrTooLarge
		}
	}
	seen := map[string]bool{}
	for i < len(parts) {
		fields := bytes.SplitN(parts[i], []byte{'\t'}, 3)
		i++
		if len(fields) != 3 {
			return nil, ErrUnavailable
		}
		p, old := string(fields[2]), ""
		if p == "" {
			if i+1 >= len(parts) {
				return nil, ErrUnavailable
			}
			old, p = string(parts[i]), string(parts[i+1])
			i += 2
		}
		index, ok := byPath[p]
		if !ok || seen[p] || out[index].OldPath != old {
			return nil, ErrUnavailable
		}
		seen[p] = true
		if string(fields[0]) == "-" && string(fields[1]) == "-" {
			continue
		}
		added, err := strconv.ParseInt(string(fields[0]), 10, 64)
		removed, other := strconv.ParseInt(string(fields[1]), 10, 64)
		if err != nil || other != nil || added < 0 || removed < 0 || added > 1<<53-1 || removed > 1<<53-1 {
			return nil, ErrUnavailable
		}
		out[index].Additions, out[index].Deletions = &added, &removed
	}
	if len(seen) != len(out) {
		return nil, ErrUnavailable
	}
	return out, nil
}
