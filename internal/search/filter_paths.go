package search

import "strings"

func compileCandidateFilter(q Query) (func(string) bool, error) {
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
	return func(relative string) bool {
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
		return included && !excluded
	}, nil
}
