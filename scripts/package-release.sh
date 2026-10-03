#!/usr/bin/env bash
# 构建同一提交的 Linux 前后端包；先 npm ci，输出目录只存发布产物。
set -euo pipefail
release_version=${1:?用法: scripts/package-release.sh VERSION [OUTPUT_DIR]}
[[ "$release_version" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$ ]] || { echo '版本名无效' >&2; exit 1; }
repo_root=$(cd "$(dirname "$0")/.." && pwd)
cd "$repo_root"
output_dir=${2:-dist/release}
mkdir -p "$output_dir"
output_dir=$(cd "$output_dir" && pwd)
npm --prefix web run build
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
for release_arch in amd64 arm64; do
  mkdir -p "$stage/persistty/bin"
  CGO_ENABLED=0 GOOS=linux GOARCH="$release_arch" go build -trimpath -buildvcs=false -o "$stage/persistty/bin/persistty" ./cmd/persistty
  cp -R web/dist "$stage/persistty/web"
  python3 - "$stage/persistty/deploy" <<'PY_COPY'
import shutil,sys
shutil.copytree('deploy',sys.argv[1],ignore=shutil.ignore_patterns('__pycache__','*.pyc','.DS_Store'))
PY_COPY
  cp LICENSE "$stage/persistty/LICENSE"
  python3 scripts/release-manifest.py "$stage/persistty" "$release_version" "$release_arch"
  python3 - "$stage" "$output_dir/persistty-linux-$release_arch.tar.gz" <<'PY_TAR'
import gzip,subprocess,sys,tarfile
from pathlib import Path
stage=Path(sys.argv[1]); epoch=int(subprocess.check_output(['git','log','-1','--format=%ct']))
def normalize(info):
    if not (info.isfile() or info.isdir()):
        raise ValueError('发布包禁止链接/特殊文件：'+info.name)
    info.uid=info.gid=0; info.uname=info.gname=''; info.mtime=epoch; info.pax_headers={}
    info.mode=0o755 if info.isdir() or info.name in ('persistty/bin/persistty','persistty/deploy/persistty-deploy.py') else 0o644
    return info
with open(sys.argv[2],'wb') as output, gzip.GzipFile(filename='',mode='wb',fileobj=output,mtime=0) as compressed:
    with tarfile.open(fileobj=compressed,mode='w',format=tarfile.PAX_FORMAT) as archive:
        archive.add(stage/'persistty',arcname='persistty',filter=normalize)
PY_TAR
  rm -rf "$stage/persistty"
done
cp deploy/persistty-deploy.py "$output_dir/persistty-deploy.py"
python3 - "$output_dir" <<'PY'
import hashlib,sys
from pathlib import Path
root=Path(sys.argv[1]); names=['persistty-linux-amd64.tar.gz','persistty-linux-arm64.tar.gz','persistty-deploy.py']
(root/'SHA256SUMS').write_text(''.join(f'{hashlib.sha256((root/name).read_bytes()).hexdigest()}  {name}\n' for name in names))
PY
printf '发布包已生成：%s\n' "$output_dir"
