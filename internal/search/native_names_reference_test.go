package search

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"persistty/internal/toolrunner"
)

// nativeNames reads only the private, already bounded placeholder tree. It never
// walks a registered root or reads ordinary source contents a second time.
func nativeNames(ctx context.Context, tree string) (map[string]bool, error) {
	allowed := map[string]bool{}
	budget := ignoreBudget{}
	outputBytes := 0
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
		rules, err = loadIgnoreRules(ctx, directory, rules, &budget, func(filename string) ([]byte, error) {
			control := filepath.Join(tree, filepath.FromSlash(path.Join(directory, filename)))
			info, err := os.Lstat(control)
			if os.IsNotExist(err) {
				return nil, nil
			}
			if err != nil {
				return nil, err
			}
			if !info.Mode().IsRegular() {
				return nil, nil
			}
			return os.ReadFile(control)
		})
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.Name() == ".git" || entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			relative := path.Join(directory, entry.Name())
			ignored, err := ignoredName(ctx, relative, entry.IsDir(), rules, &budget)
			if err != nil {
				return err
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
