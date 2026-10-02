package search

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"persistty/internal/toolrunner"
)

type nameIgnore struct {
	scope              string
	pattern            *regexp.Regexp
	negated, directory bool
}

// nativeNames reads only the private, already bounded placeholder tree. It never
// walks a registered root or reads ordinary source contents a second time.
func nativeNames(ctx context.Context, tree string) (map[string]bool, error) {
	allowed := map[string]bool{}
	rulesCount, comparisons, outputBytes := 0, 0, 0
	var walk func(string, [4][]nameIgnore) error
	walk = func(directory string, rules [4][]nameIgnore) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, err := os.ReadDir(filepath.Join(tree, filepath.FromSlash(directory)))
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.Name() == ".git" && entry.IsDir() {
				// A nested repository's Git rules do not inherit outer Git rules.
				rules[0], rules[1] = nil, nil
				break
			}
		}
		for rank, filename := range []string{".git/info/exclude", ".gitignore", ".ignore", ".rgignore"} {
			control := filepath.Join(tree, filepath.FromSlash(path.Join(directory, filename)))
			info, err := os.Lstat(control)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				continue
			}
			content, err := os.ReadFile(control)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			// Force child append to allocate, keeping sibling rule stacks independent.
			rules[rank] = rules[rank][:len(rules[rank]):len(rules[rank])]
			for _, line := range strings.Split(string(content), "\n") {
				if err := ctx.Err(); err != nil {
					return err
				}
				rule, ok := parseNameIgnore(directory, strings.TrimSuffix(line, "\r"))
				if !ok {
					continue
				}
				rulesCount++
				if rulesCount > 10000 {
					return toolrunner.ErrLimit
				}
				rules[rank] = append(rules[rank], rule)
			}
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.Name() == ".git" || entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			relative := path.Join(directory, entry.Name())
			ignored, matched := false, false
			// File type priority is independent of depth: rgignore > ignore > gitignore > exclude.
			for rank := len(rules) - 1; rank >= 0 && !matched; rank-- {
				for i := len(rules[rank]) - 1; i >= 0; i-- {
					if err := ctx.Err(); err != nil {
						return err
					}
					comparisons++
					if comparisons > 2000000 {
						return toolrunner.ErrLimit
					}
					rule := rules[rank][i]
					if rule.directory && !entry.IsDir() {
						continue
					}
					local := relative
					if rule.scope != "" {
						local = strings.TrimPrefix(relative, rule.scope+"/")
					}
					if rule.pattern.MatchString(local) {
						ignored, matched = !rule.negated, true
						break
					}
				}
			}
			if ignored {
				continue
			}
			if entry.IsDir() {
				if err := walk(relative, rules); err != nil {
					return err
				}
			} else if entry.Type().IsRegular() {
				outputBytes += len(relative) + 1
				if outputBytes > 32<<20 {
					return toolrunner.ErrLimit
				}
				allowed[relative] = true
			}
		}
		return nil
	}
	return allowed, walk("", [4][]nameIgnore{})
}

func parseNameIgnore(scope, line string) (nameIgnore, bool) {
	// Git trims unescaped trailing spaces, while preserving escaped spaces.
	for strings.HasSuffix(line, " ") {
		backslashes := 0
		for i := len(line) - 2; i >= 0 && line[i] == '\\'; i-- {
			backslashes++
		}
		if backslashes%2 == 1 {
			break
		}
		line = strings.TrimSuffix(line, " ")
	}
	if line == "" || strings.HasPrefix(line, "#") {
		return nameIgnore{}, false
	}
	rule := nameIgnore{scope: scope, negated: strings.HasPrefix(line, "!")}
	if rule.negated {
		line = strings.TrimPrefix(line, "!")
	}
	rule.directory = strings.HasSuffix(line, "/")
	line = strings.TrimSuffix(line, "/")
	anchored := strings.Contains(line, "/")
	line = strings.TrimPrefix(line, "/")
	if line == "" {
		return nameIgnore{}, false
	}
	var expression strings.Builder
	braces := 0
	if anchored {
		expression.WriteString("^")
	} else {
		expression.WriteString("(^|/)")
	}
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
			if i == len(line) {
				return nameIgnore{}, false
			}
			expression.WriteString(regexp.QuoteMeta(line[i : i+1]))
		case '*':
			end := i
			for end+1 < len(line) && line[end+1] == '*' {
				end++
			}
			if end > i && (i == 0 || line[i-1] == '/') && (end+1 == len(line) || line[end+1] == '/') {
				if end+1 < len(line) {
					expression.WriteString("(.*/)?")
					end++
				} else {
					expression.WriteString(".*")
				}
			} else {
				expression.WriteString("[^/]*")
			}
			i = end
		case '?':
			expression.WriteString("[^/]")
		case '{':
			braces++
			expression.WriteString("(?:")
		case '}':
			if braces == 0 {
				expression.WriteString("\\}")
			} else {
				braces--
				expression.WriteString(")")
			}
		case ',':
			if braces > 0 {
				expression.WriteString("|")
			} else {
				expression.WriteString(",")
			}
		case '[':
			end := i + 1
			if end < len(line) && (line[end] == '!' || line[end] == '^') {
				end++
			}
			if end < len(line) && line[end] == ']' {
				end++
			}
			for end < len(line) && line[end] != ']' {
				if line[end] == '\\' {
					end++
				}
				end++
			}
			if end >= len(line) {
				return nameIgnore{}, false
			}
			// rg/globset treats a nested [ literally, not as a POSIX class.
			class := "[" + strings.ReplaceAll(line[i+1:end+1], "[", `\[`)
			if strings.HasPrefix(class, "[!") {
				class = "[^" + class[2:]
			}
			expression.WriteString(class)
			i = end
		default:
			expression.WriteString(regexp.QuoteMeta(line[i : i+1]))
		}
	}
	if braces != 0 {
		return nameIgnore{}, false
	}
	expression.WriteString("$")
	pattern, err := regexp.Compile(expression.String())
	rule.pattern = pattern
	return rule, err == nil
}
