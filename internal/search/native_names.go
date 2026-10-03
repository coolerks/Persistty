package search

import (
	"context"
	"regexp"
	"strings"

	"persistty/internal/toolrunner"
)

type nameIgnore struct {
	scope              string
	pattern            *regexp.Regexp
	negated, directory bool
}

// The discovery walker and staged compatibility checks share one ignore owner.
type ignoreRules [4][]nameIgnore
type ignoreBudget struct{ rules, comparisons int }

func loadIgnoreRules(ctx context.Context, directory string, rules ignoreRules, budget *ignoreBudget, load func(string) ([]byte, error)) (ignoreRules, error) {
	for rank, filename := range []string{".git/info/exclude", ".gitignore", ".ignore", ".rgignore"} {
		content, err := load(filename)
		if err != nil {
			return rules, err
		}
		rules[rank] = rules[rank][:len(rules[rank]):len(rules[rank])]
		for _, line := range strings.Split(string(content), "\n") {
			if err := ctx.Err(); err != nil {
				return rules, err
			}
			rule, ok := parseNameIgnore(directory, strings.TrimSuffix(line, "\r"))
			if !ok {
				continue
			}
			budget.rules++
			if budget.rules > 10000 {
				return rules, toolrunner.ErrLimit
			}
			rules[rank] = append(rules[rank], rule)
		}
	}
	return rules, nil
}

func ignoredName(ctx context.Context, relative string, directory bool, rules ignoreRules, budget *ignoreBudget) (bool, error) {
	for rank := len(rules) - 1; rank >= 0; rank-- {
		for i := len(rules[rank]) - 1; i >= 0; i-- {
			if err := ctx.Err(); err != nil {
				return false, err
			}
			budget.comparisons++
			if budget.comparisons > 2000000 {
				return false, toolrunner.ErrLimit
			}
			rule := rules[rank][i]
			if rule.directory && !directory {
				continue
			}
			local := relative
			if rule.scope != "" {
				local = strings.TrimPrefix(relative, rule.scope+"/")
			}
			if rule.pattern.MatchString(local) {
				return !rule.negated, nil
			}
		}
	}
	return false, nil
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
