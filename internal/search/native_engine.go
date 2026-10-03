package search

import (
	"context"
	"errors"
	"persistty/internal/toolrunner"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A search snapshot pins its engine, so preview never silently changes the
// interpretation of captures after a tool installation or removal.
func selectEngine(ctx context.Context, runner *toolrunner.Runner, dir string, q Query) (bool, error) {
	if !q.Regex {
		_, err := nativePattern(q)
		return true, err
	}
	_, err := rgEngine(ctx, runner, dir, q, "", nil)
	if err == nil {
		err = rgReplacementCapability(ctx, runner, dir)
	}
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, toolrunner.ErrUnavailable) && !errors.Is(err, ErrPattern) {
		return false, err
	}
	_, nativeErr := nativePattern(q)
	return true, nativeErr
}

// rg before 15 accepts --replace with --json but omits the expanded captures.
// Probe a known match before pinning the engine, including for searches whose
// pattern does not match the probe. Preview must never switch engines later.
func rgReplacementCapability(ctx context.Context, runner *toolrunner.Runner, dir string) error {
	replacement := "<$1>$$"
	matches, err := rgEngine(ctx, runner, dir, Query{Pattern: "(a)", Regex: true, CaseSensitive: true}, "a\n", &replacement)
	if err != nil {
		return err
	}
	if len(matches) != 1 || matches[0].Replacement != "<a>$" {
		return toolrunner.ErrUnavailable
	}
	return nil
}

func engine(ctx context.Context, runner *toolrunner.Runner, dir string, q Query, content string, replacement *string, native bool) ([]expanded, error) {
	if native {
		return nativeEngine(ctx, q, content, replacement)
	}
	return rgEngine(ctx, runner, dir, q, content, replacement)
}

func nativePattern(q Query) (*regexp.Regexp, error) {
	pattern := q.Pattern
	if !q.Regex {
		pattern = regexp.QuoteMeta(pattern)
	}
	if !q.CaseSensitive {
		pattern = "(?i:" + pattern + ")"
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, ErrPattern
	}
	return re, nil
}

func wordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsNumber(r) || unicode.In(r, unicode.Pc) || r == '\u200c' || r == '\u200d'
}

func utf16Length(text string) int {
	units := 0
	for _, r := range text {
		units++
		if r > 0xffff {
			units++
		}
	}
	return units
}

func nativeEngine(ctx context.Context, q Query, content string, replacement *string) ([]expanded, error) {
	re, err := nativePattern(q)
	if err != nil {
		return nil, err
	}
	return nativeMatches(ctx, q, re, content, replacement)
}

func nativeMatches(ctx context.Context, q Query, re *regexp.Regexp, content string, replacement *string) ([]expanded, error) {
	result := []expanded{}
	// Match each physical line while retaining byte offsets into the original
	// snapshot. BOM and every newline byte survive preview/apply unchanged.
	for offset, number := 0, 1; offset < len(content); number++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := offset + strings.IndexAny(content[offset:], "\r\n")
		if end < offset {
			end = len(content)
		}
		line := content[offset:end]
		preview := line
		if len(preview) > 2048 {
			count := 0
			for index := range preview {
				if count == 512 {
					preview = preview[:index]
					break
				}
				count++
			}
		}
		column, measured := 1, 0
		if number == 1 && strings.HasPrefix(line, "\ufeff") {
			measured = len("\ufeff")
		}
		// Bound temporary match storage even for a single multi-megabyte line.
		indexesList := re.FindAllStringSubmatchIndex(line, 5001)
		for _, indexes := range indexesList {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			start, finish := indexes[0], indexes[1]
			if q.WholeWord {
				before, _ := utf8.DecodeLastRuneInString(line[:start])
				after, _ := utf8.DecodeRuneInString(line[finish:])
				if start > 0 && wordRune(before) || finish < len(line) && wordRune(after) {
					continue
				}
			}
			if start >= measured {
				column += utf16Length(line[measured:start])
				measured = start
			}
			m := Match{Line: number, Column: column, EndColumn: column + utf16Length(line[start:finish]), Preview: preview, Start: offset + start, End: offset + finish}
			value := ""
			if replacement != nil {
				value = *replacement
				if q.Regex {
					value = string(re.ExpandString(nil, value, line, indexes))
				}
			}
			result = append(result, expanded{m, value})
			if len(result) > 5000 {
				return result, nil
			}
		}
		if len(indexesList) == 5001 {
			// Whole-word filtering may reject the first batch. Do not return an
			// apparently complete empty result while later matches are unexamined.
			return nil, toolrunner.ErrLimit
		}
		if end == len(content) {
			break
		}
		offset = end + 1
		if content[end] == '\r' && offset < len(content) && content[offset] == '\n' {
			offset++
		}
	}
	return result, ctx.Err()
}
