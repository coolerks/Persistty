#!/usr/bin/env python3
"""发布包的实际提交、迁移校验与后端第三方许可。"""
import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path


def command(*args):
    return subprocess.check_output(args, text=True).strip()


def main():
    dest, version, arch = Path(sys.argv[1]), sys.argv[2], sys.argv[3]
    migrations = [{"version": int(path.name.split('_')[0]), "checksum": hashlib.sha256(path.read_bytes()).hexdigest()}
                  for path in sorted(Path('internal/storage/migrations').glob('*.sql'))]
    metadata = {"format": 1, "version": version, "commit": command('git', 'rev-parse', 'HEAD'),
                "os": "linux", "arch": arch, "go": command('go', 'version'), "migrations": migrations}
    (dest/'version.json').write_text(json.dumps(metadata, indent=2)+'\n')
    os.environ.update(GOOS='linux', GOARCH=arch, CGO_ENABLED='0')
    modules = command('go', 'list', '-deps', '-f', '{{if .Module}}{{.Module.Path}}|{{.Module.Version}}|{{.Module.Dir}}{{end}}', './cmd/persistty')
    notices = []
    for module in sorted(set(modules.splitlines())):
        if not module.strip():
            continue
        name, module_version, directory = module.split('|')
        if name == 'persistty':
            continue
        files = sorted(p for p in Path(directory).iterdir() if p.is_file() and re.match(r'^(license|licence|copying|notice)(\.|$)', p.name, re.I))
        if not any(re.match(r'^(license|licence|copying)(\.|$)', p.name, re.I) for p in files):
            raise SystemExit(f'后端依赖缺少许可，拒绝发布：{name}@{module_version}')
        notices.append(f'{name}@{module_version}\n'+''.join(f'\n{p.name}\n{p.read_text()}\n' for p in files))
    (dest/'backend-third-party-licenses.txt').write_text('\n'.join(notices))


if __name__ == '__main__':
    main()
