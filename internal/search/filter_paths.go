package search

import (
	"context"
	"errors"
	"persistty/internal/toolrunner"
	"strings"
)

func (s *Service) filterPaths(ctx context.Context, tree string, q Query, allowed map[string]bool) ([]byte, error) {
	if len(q.Include)+len(q.Exclude) == 0 {
		return []byte(strings.Join(sortedKeys(allowed), "\x00")), nil
	}
	args := []string{"--no-config", "--no-require-git", "--files", "--hidden", "--null"}
	for _, glob := range q.Include {
		args = append(args, "--glob", glob)
	}
	for _, glob := range q.Exclude {
		args = append(args, "--glob", "!"+glob)
	}
	args = append(args, "--glob", "!.git/**", "--glob", "!**/.git/**", "--", ".")
	out, code, err := s.Runner.Run(ctx, "rg", tree, nil, args...)
	if err == nil && (code == 0 || code == 1) {
		return out, nil
	}
	if err != nil && !errors.Is(err, toolrunner.ErrUnavailable) {
		return nil, err
	}
	// Query globs only narrow the already ignored safe candidates. They can
	// never whitelist files excluded by discovery.
	include, exclude := []nameIgnore{}, []nameIgnore{}
	for i, globs := range [][]string{q.Include, q.Exclude} {
		for _, glob := range globs {
			rule, valid := parseNameIgnore("", glob)
			if !valid {
				return nil, ErrInvalid
			}
			if i == 0 {
				include = append(include, rule)
			} else {
				exclude = append(exclude, rule)
			}
		}
	}
	paths := []string{}
	for _, relative := range sortedKeys(allowed) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		included := len(include) == 0
		for _, rule := range include {
			if !rule.directory && rule.pattern.MatchString(relative) {
				included = true
			}
		}
		excluded := false
		for _, rule := range exclude {
			for candidate := relative; candidate != ""; {
				if (candidate != relative || !rule.directory) && rule.pattern.MatchString(candidate) {
					excluded = true
					break
				}
				index := strings.LastIndexByte(candidate, '/')
				if index < 0 {
					break
				}
				candidate = candidate[:index]
			}
		}
		if included && !excluded {
			paths = append(paths, relative)
		}
	}
	return []byte(strings.Join(paths, "\x00")), nil
}
