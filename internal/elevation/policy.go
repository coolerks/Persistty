package elevation

import (
	"io"
	"path/filepath"
	"strings"

	"persistty/internal/storage"
)

type Policy struct {
	Schema         int      `json:"schema"`
	CallerUID      int      `json:"caller_uid"`
	CallerGID      int      `json:"caller_gid"`
	LedgerPath     string   `json:"ledger_path"`
	ProtectedPaths []string `json:"protected_paths"`
	Targets        []Target `json:"targets"`
}

func ParsePolicy(data []byte) (Policy, error) {
	var p Policy
	if len(data) > 64<<10 || strictJSON(data, &p) != nil {
		return p, ErrInvalid
	}
	if p.Schema != 1 || p.CallerUID <= 0 || p.CallerGID < 0 || !validAbsolute(p.LedgerPath) || p.LedgerPath == "/" || len(p.Targets) < 1 || len(p.Targets) > 128 || len(p.ProtectedPaths) > 128 {
		return p, ErrInvalid
	}
	ids, paths := map[string]bool{}, map[string]bool{}
	for _, path := range p.ProtectedPaths {
		if !validAbsolute(path) {
			return p, ErrInvalid
		}
	}
	for _, target := range p.Targets {
		if !targetPattern.MatchString(target.ID) || !validAbsolute(target.Path) || target.Path == "/" || ids[target.ID] || paths[target.Path] || p.protected(target.Path) {
			return p, ErrInvalid
		}
		ids[target.ID] = true
		paths[target.Path] = true
	}
	if p.protectedInfrastructure(p.LedgerPath) {
		return p, ErrInvalid
	}
	return p, nil
}
func within(path, base string) bool {
	return path == base || strings.HasPrefix(path, strings.TrimSuffix(base, "/")+"/")
}
func (p Policy) protectedInfrastructure(path string) bool {
	for _, base := range []string{"/etc/persistty-elevation", HelperPath, BrokerPath, "/usr/bin/sudo", "/usr/local/bin/persistty", "/etc/persistty", "/etc/sudoers", "/etc/sudoers.d", "/etc/pam.d", "/etc/passwd", "/etc/shadow", "/etc/group", "/etc/gshadow", "/etc/security", "/etc/ssh", "/root/.ssh", "/etc/login.defs", "/etc/nsswitch.conf", "/etc/ld.so.conf", "/etc/ld.so.conf.d", "/etc/ld.so.preload", "/etc/systemd", "/usr/lib"} {
		if within(path, base) {
			return true
		}
	}
	return false
}
func (p Policy) protected(path string) bool {
	if p.protectedInfrastructure(path) || within(path, p.LedgerPath) {
		return true
	}
	for _, base := range p.ProtectedPaths {
		if within(path, base) {
			return true
		}
	}
	return false
}
func LoadPolicy() (Policy, error) {
	file, err := openTrusted(PolicyPath, false)
	if err != nil {
		return Policy{}, ErrUnavailable
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() > 64<<10 {
		return Policy{}, ErrUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(file, 64<<10+1))
	if err != nil {
		return Policy{}, ErrUnavailable
	}
	p, err := ParsePolicy(data)
	if err != nil {
		return Policy{}, ErrUnavailable
	}
	return p, nil
}
func (p Policy) target(root storage.RegisteredFolder, relative string) (Target, error) {
	if !validAbsolute(root.Path) || !validRelativeTarget(relative) || root.ProjectVersion < 1 {
		return Target{}, ErrInvalid
	}
	absolute := filepath.Join(root.Path, relative)
	for _, target := range p.Targets {
		if target.Path == absolute {
			return target, nil
		}
	}
	return Target{}, ErrForbidden
}
func (p Policy) validateGrant(g Grant) error {
	if err := g.Validate(timeNow()); err != nil {
		return err
	}
	target, err := p.target(g.Root, g.Path)
	if err != nil {
		return err
	}
	if target.ID != g.TargetID {
		return ErrForbidden
	}
	return nil
}
