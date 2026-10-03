#!/usr/bin/env python3
"""Persistty Debian 安装/更新器。仅依赖 Python 标准库；详见 deploy/README.md。"""
import argparse
import contextlib
import fcntl
import hashlib
import ipaddress
import json
import os
import platform
import pwd
import re
import shutil
import sqlite3
import stat
import struct
import subprocess
import sys
import tarfile
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path, PurePosixPath

VERSION = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,79}\Z")
REPO = re.compile(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+\Z")
ROOT = Path('/opt/persistty')
STATE = Path('/etc/persistty-deploy/state.json')
CONFIG = Path('/etc/persistty/config.yaml')
DATA = Path('/var/lib/persistty')
BACKUPS = Path('/var/backups/persistty')
LOCK = Path('/run/lock/persistty-deploy.lock')
MAX_ARCHIVE = 512 * 1024 * 1024
MAX_EXPANDED = 1024 * 1024 * 1024
MAX_FILES = 10000
GITHUB_HOSTS = {'api.github.com', 'github.com', 'release-assets.githubusercontent.com',
                'objects.githubusercontent.com'}


class DeployError(Exception):
    pass


def require(condition, message):
    if not condition:
        raise DeployError(message)


def lan_host(value):
    """仅允许 RFC1918 IPv4，避免把部署参数直接拼成配置指令。"""
    try:
        address = ipaddress.IPv4Address(value)
    except (ValueError, TypeError):
        raise DeployError('--host 需要局域网 IPv4 地址') from None
    require(any(address in ipaddress.IPv4Network(network) for network in
                ('10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16')),
            '--host 需要局域网私有 IPv4 地址')
    return str(address)


def host_from_settings(settings):
    origin = settings.get('origin', '')
    require(isinstance(origin, str) and origin.startswith('http://'), '部署记录需要局域网 HTTP 地址')
    host = lan_host(origin[len('http://'):])
    require(origin == 'http://' + host, '部署记录的地址格式不正确')
    return host


def private_file(path, uid=0):
    """配置/token 不接受符号链接、硬链接或宽松权限。"""
    info = path.lstat()
    require(stat.S_ISREG(info.st_mode) and info.st_uid == uid and info.st_nlink == 1
            and not info.st_mode & 0o077, f'私有文件所有权/权限不符合要求：{path}')
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    try:
        require(os.fstat(fd) == info, f'文件在检查后发生变化：{path}')
        with os.fdopen(fd, 'rb', closefd=False) as source:
            data = source.read(65537)
        require(len(data) <= 65536, f'私有文件过大：{path}')
        return data
    finally:
        os.close(fd)


def managed_dir(path, mode=0o755, uid=0, gid=0):
    """仅对安装器拥有的叶目录操作；禁止任何父目录符号链接。"""
    for parent in reversed(path.parents):
        info = parent.lstat()
        require(stat.S_ISDIR(info.st_mode) and info.st_uid in (0, uid) and not info.st_mode & 0o022,
                f'父目录必须是真实、不可全局写入的目录：{parent}')
    if path.exists() or path.is_symlink():
        info = path.lstat()
        require(stat.S_ISDIR(info.st_mode) and info.st_uid == uid and
                stat.S_IMODE(info.st_mode) == mode, f'目录所有权/权限不符合要求：{path}')
    else:
        path.mkdir(mode=mode)
        os.chown(path, uid, gid)
        path.chmod(mode)


def atomic_write(path, data, mode=0o600, uid=0, gid=0):
    fd, temporary = tempfile.mkstemp(prefix='.' + path.name + '.', dir=path.parent)
    try:
        with os.fdopen(fd, 'wb') as dest:
            dest.write(data if isinstance(data, bytes) else data.encode())
            dest.flush()
            os.fchmod(dest.fileno(), mode)
            os.fchown(dest.fileno(), uid, gid)
            os.fsync(dest.fileno())
        os.replace(temporary, path)
        dir_fd = os.open(path.parent, os.O_DIRECTORY)
        try:
            os.fsync(dir_fd)
        finally:
            os.close(dir_fd)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


@contextlib.contextmanager
def deployment_lock(path=LOCK):
    fd = os.open(path, os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
    try:
        info = os.fstat(fd)
        require(stat.S_ISREG(info.st_mode) and info.st_uid == 0 and info.st_nlink == 1
                and stat.S_IMODE(info.st_mode) == 0o600, '部署锁文件权限不正确')
        try:
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise DeployError('另一个部署正在执行，请稍后重试') from exc
        yield
    finally:
        os.close(fd)


def github_url(url):
    parsed = urllib.parse.urlsplit(url)
    require(parsed.scheme == 'https' and parsed.hostname in GITHUB_HOSTS and
            not parsed.username and not parsed.password and parsed.port in (None, 443),
            '拒绝非 GitHub HTTPS 下载地址')
    return parsed


class Redirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, fp, code, msg, headers, newurl):
        target = github_url(newurl)
        redirected = super().redirect_request(request, fp, code, msg, headers, newurl)
        if redirected and target.hostname != urllib.parse.urlsplit(request.full_url).hostname:
            redirected.remove_header('Authorization')
        return redirected


class GitHub:
    def __init__(self, repo, token_file=None):
        require(REPO.fullmatch(repo), '仓库应为 OWNER/REPO')
        self.repo = repo
        self.token = None
        if token_file:
            token_path = Path(token_file)
            require(token_path.is_absolute(), 'token 文件必须使用绝对路径')
            self.token = private_file(token_path).decode().strip()
            require(re.fullmatch(r'[A-Za-z0-9_]+', self.token), 'token 文件格式不正确')
        self.opener = urllib.request.build_opener(Redirects())

    def fetch(self, url, limit, destination=None, asset=False):
        host = github_url(url).hostname
        headers = {'User-Agent': 'persistty-deploy', 'Accept': 'application/octet-stream' if asset
                   else 'application/vnd.github+json', 'X-GitHub-Api-Version': '2022-11-28'}
        if self.token and host == 'api.github.com':
            headers['Authorization'] = 'Bearer ' + self.token
        try:
            with self.opener.open(urllib.request.Request(url, headers=headers), timeout=60) as response:
                total = 0
                chunks = []
                while True:
                    chunk = response.read(1024 * 1024)
                    if not chunk:
                        break
                    total += len(chunk)
                    require(total <= limit, '下载内容超过大小限制')
                    if destination:
                        destination.write(chunk)
                    else:
                        chunks.append(chunk)
                return b''.join(chunks)
        except urllib.error.HTTPError as exc:
            raise DeployError(f'GitHub 返回 HTTP {exc.code}；检查版本、仓库权限或 API 限额') from None
        except (urllib.error.URLError, TimeoutError, OSError):
            raise DeployError('GitHub 下载失败；检查网络后重试（现有服务尚未切换）') from None

    def download(self, version, arch, directory):
        require(version == 'latest' or VERSION.fullmatch(version), '版本名称不正确')
        suffix = 'latest' if version == 'latest' else 'tags/' + urllib.parse.quote(version, safe='')
        release = json.loads(self.fetch(f'https://api.github.com/repos/{self.repo}/releases/{suffix}', 2 * 1024 * 1024))
        tag = release.get('tag_name', '')
        require(VERSION.fullmatch(tag) and not release.get('draft') and not release.get('prerelease'),
                '发布版本不可部署')
        require(version == 'latest' or tag == version, '返回的发布版本不匹配')
        names = [f'persistty-linux-{arch}.tar.gz', 'SHA256SUMS']
        assets = {}
        for asset in release.get('assets', []):
            if asset.get('name') in names:
                require(asset['name'] not in assets, '发布包含重复资产')
                expected_url = f'https://api.github.com/repos/{self.repo}/releases/assets/{asset.get("id")}'
                require(asset.get('url') == expected_url and isinstance(asset.get('id'), int), '资产身份不正确')
                assets[asset['name']] = expected_url
        require(set(assets) == set(names), '发布缺少安装包或 SHA256SUMS')
        sums = self.fetch(assets['SHA256SUMS'], 65536, asset=True).decode('ascii')
        checksums = parse_checksums(sums)
        name = names[0]
        require(name in checksums, 'SHA256SUMS 未包含当前架构')
        archive = directory / name
        with archive.open('xb') as output:
            self.fetch(assets[name], MAX_ARCHIVE, output, asset=True)
        hasher = hashlib.sha256()
        with archive.open('rb') as source:
            for chunk in iter(lambda: source.read(1024 * 1024), b''):
                hasher.update(chunk)
        digest = hasher.hexdigest()
        require(digest == checksums[name], '安装包 SHA-256 校验失败；现有服务未改变')
        extracted = directory / 'extracted'
        extracted.mkdir()
        extracted.chmod(0o755)
        safe_extract(archive, extracted)
        package = extracted / 'persistty'
        manifest = validate_package(package, tag, arch)
        return package, manifest, digest


def parse_checksums(text):
    results = {}
    for line in text.splitlines():
        match = re.fullmatch(r'([a-f0-9]{64}) [ *]([A-Za-z0-9._-]+)', line)
        require(match is not None, 'SHA256SUMS 格式不正确')
        checksum, name = match.groups()
        require(name not in results, 'SHA256SUMS 包含重复文件')
        results[name] = checksum
    return results


def safe_extract(archive, destination):
    total, seen = 0, set()
    with tarfile.open(archive, 'r:gz') as source:
        for count, member in enumerate(source, 1):
            raw = member.name.rstrip('/')
            path = PurePosixPath(raw)
            require(raw and not raw.startswith('/') and '..' not in path.parts and
                    path.parts[0] == 'persistty' and str(path) == raw and '\\' not in raw,
                    '安装包路径不安全')
            require(raw not in seen and count <= MAX_FILES, '安装包重复文件或文件数量超限')
            seen.add(raw)
            require(member.isdir() or member.isfile(), '安装包禁止链接和特殊文件')
            total += member.size
            require(0 <= member.size <= MAX_EXPANDED and total <= MAX_EXPANDED, '安装包解压大小超限')
            target = destination / path
            if member.isdir():
                target.mkdir(parents=True, exist_ok=True)
                target.chmod(0o755)
            else:
                target.parent.mkdir(parents=True, exist_ok=True)
                with source.extractfile(member) as inp, target.open('xb') as out:
                    shutil.copyfileobj(inp, out)
                target.chmod(0o755 if raw in ('persistty/bin/persistty', 'persistty/deploy/persistty-deploy.py') else 0o644)


def validate_package(package, version, arch):
    manifest = json.loads((package / 'version.json').read_text())
    require(manifest.get('format') == 1 and manifest.get('version') == version and
            manifest.get('os') == 'linux' and manifest.get('arch') == arch and
            re.fullmatch('[a-f0-9]{40}', manifest.get('commit', '')), '安装包版本/平台清单不正确')
    migrations = manifest.get('migrations')
    require(isinstance(migrations, list) and bool(migrations), '安装包缺少数据库迁移清单')
    versions = set()
    for migration in migrations:
        v = migration.get('version')
        require(isinstance(v, int) and v > 0 and v not in versions and
                re.fullmatch('[a-f0-9]{64}', migration.get('checksum', '')), '迁移清单不正确')
        versions.add(v)
    required = ('bin/persistty', 'web/index.html', 'web/third-party-licenses.txt',
                'LICENSE', 'backend-third-party-licenses.txt', 'deploy/config.example.yaml',
                'deploy/persistty-deploy.py', 'deploy/tmux.example.conf',
                'deploy/systemd/persistty.service', 'deploy/systemd/persistty-tmux.service',
                'deploy/nginx/persistty.conf')
    require(all((package / name).is_file() for name in required), '安装包不完整')
    with (package / 'bin/persistty').open('rb') as source:
        header = source.read(64)
    require(len(header) == 64 and header[:6] == b'\x7fELF\x02\x01' and
            struct.unpack_from('<H', header, 18)[0] == {'amd64': 62, 'arm64': 183}[arch], '后端不是匹配架构的 ELF64 二进制')
    return manifest


def write_path(value):
    path = Path(value)
    require(path.is_absolute() and path.is_dir() and str(path.resolve()) == str(path) and
            not any(c in value for c in '\n\r\t\x00') and value not in ('/', '/etc', '/usr', '/opt', '/var'),
            '项目可写目录必须是真实、已存在的绝对目录，不能为系统根目录')
    return str(path)


def unit_path(value):
    return '"' + value.replace('%', '%%').replace('\\', '\\\\').replace('"', '\\"') + '"'


def run(*args, capture=False):
    result = subprocess.run(args, stdout=subprocess.PIPE if capture else None,
                            stderr=subprocess.PIPE if capture else None, text=True, check=False)
    require(result.returncode == 0, '系统命令失败：' + args[0] + ' ' + ' '.join(args[1:3]))
    return result.stdout.strip() if capture else None


def backup_database(database, output):
    if not database.exists():
        return False
    require(stat.S_ISREG(database.lstat().st_mode), '数据库不得是链接或特殊文件')
    require(not output.exists(), '备份文件已经存在')
    deadline = time.monotonic() + 90
    def progress(_status, _remaining, _total):
        require(time.monotonic() < deadline, 'SQLite 备份超时，拒绝切换代码')
    try:
        with contextlib.closing(sqlite3.connect(database.as_uri() + '?mode=ro', uri=True, timeout=30)) as source:
            with contextlib.closing(sqlite3.connect(output)) as destination:
                source.backup(destination, pages=1024, progress=progress)
                result = destination.execute('PRAGMA quick_check').fetchone()[0]
                require(result == 'ok', 'SQLite 备份一致性检查失败')
        output.chmod(0o600)
    except BaseException:
        if output.exists():
            output.unlink()
        raise
    return True


def compatible_database(database, manifest):
    if not database.exists():
        return True
    try:
        with contextlib.closing(sqlite3.connect(database.as_uri() + '?mode=ro', uri=True)) as source:
            rows = source.execute('SELECT version, checksum FROM schema_migrations').fetchall()
        expected = {item['version']: item['checksum'] for item in manifest['migrations']}
        return all(expected.get(version) == checksum for version, checksum in rows)
    except (sqlite3.Error, KeyError):
        return False


def switch_current(root, target):
    temporary = root / '.current-new'
    require(not temporary.exists() and not temporary.is_symlink(), '遗留的 current 临时链接需要人工检查')
    temporary.symlink_to(target)
    os.replace(temporary, root / 'current')


def api_healthy(url):
    try:
        urllib.request.urlopen(url, timeout=3).close()
        return False
    except urllib.error.HTTPError as exc:
        if exc.code != 401:
            return False
        try:
            return json.loads(exc.read(8192)).get('error', {}).get('code') == 'unauthenticated'
        except (ValueError, AttributeError):
            return False
        finally:
            exc.close()
    except (OSError, urllib.error.URLError):
        return False


class Deployment:
    def __init__(self, settings, root=ROOT, data=DATA, backups=BACKUPS):
        self.settings, self.root, self.data, self.backups = settings, root, data, backups

    def healthy(self):
        result = subprocess.run(['systemctl', 'is-active', '--quiet', 'persistty.service'], check=False)
        return result.returncode == 0 and api_healthy('http://127.0.0.1:8080/api/v1/auth/session')

    def wait_healthy(self):
        for _ in range(30):
            if self.healthy():
                return
            time.sleep(1)
        raise DeployError('新版本启动检查失败；查看 journalctl -u persistty.service')

    def proxy_healthy(self):
        origin = self.settings['origin']
        require(api_healthy(origin + '/api/v1/auth/session'), 'Nginx API 检查失败；确认 http 配置已 include /opt/persistty/nginx.conf')
        try:
            with urllib.request.urlopen(origin + '/login', timeout=5) as response:
                require(response.status == 200 and b'<html' in response.read(65536).lower(), 'Nginx 登录页检查失败')
        except (OSError, urllib.error.URLError):
            raise DeployError('Nginx 登录页检查失败；检查 ' + origin + ' 监听与 include 配置') from None

    def activate(self, package, manifest, digest):
        releases = self.root / 'releases'
        target = releases / manifest['version']
        receipt = {'sha256': digest, 'repo': self.settings['repo']}
        if target.exists():
            require(target.is_dir() and not target.is_symlink() and
                    json.loads((target / '.receipt.json').read_text()) == receipt,
                    '已有同版本内容不同，拒绝覆盖')
        else:
            require(not target.is_symlink(), '版本目录不能为符号链接')
            staging = Path(tempfile.mkdtemp(prefix='.release-', dir=releases))
            try:
                shutil.copytree(package, staging, dirs_exist_ok=True)
                staging.chmod(0o755)
                atomic_write(staging / '.receipt.json', json.dumps(receipt), 0o644)
                os.rename(staging, target)
            finally:
                if staging.exists():
                    shutil.rmtree(staging)
        current = self.root / 'current'
        require(not current.exists() or current.is_symlink(), 'current 必须为安装器管理的链接')
        old = current.resolve() if current.is_symlink() else None
        if old:
            require(old.parent == releases.resolve() and old.is_dir(), 'current 指向不受管理的目录')
        if old == target and self.healthy():
            self.proxy_healthy()
            print('已是同一版本且健康，无需重启：' + manifest['version'])
            return target
        backup = self.backups / (time.strftime('%Y%m%dT%H%M%SZ', time.gmtime()) + '-' + manifest['version'] + '-' + str(time.time_ns()) + '.db')
        print('包已校验，停止 Web 并创建一致备份；tmux 保持运行。')
        run('systemctl', 'stop', 'persistty.service')
        switched = False
        try:
            if backup_database(self.data / 'metadata.db', backup):
                print('数据库备份：' + str(backup))
            switch_current(self.root, target)
            switched = True
            run('systemctl', 'start', 'persistty.service')
            self.wait_healthy()
            self.proxy_healthy()
        except BaseException:
            if switched:
                run('systemctl', 'stop', 'persistty.service')
                if old:
                    switch_current(self.root, old)
            if old:
                old_manifest = json.loads((old / 'version.json').read_text())
                if compatible_database(self.data / 'metadata.db', old_manifest):
                    run('systemctl', 'start', 'persistty.service')
                    self.wait_healthy()
                    print('部署失败，旧代码已恢复；数据库未覆盖。', file=sys.stderr)
                else:
                    print('数据库与旧代码不兼容，Web 保持停止。备份：' + str(backup) +
                          '；按部署文档人工恢复或部署兼容的新版本。', file=sys.stderr)
            else:
                print('首次启动失败，Web 保持停止；修正配置后执行 update 重试。', file=sys.stderr)
            raise
        print('已部署：' + manifest['version'] + ' / ' + manifest['commit'][:12])
        return target


def find_program(name, explicit=None):
    candidates = [explicit] if explicit else [shutil.which(name), f'/usr/local/bin/{name}',
        f'/usr/bin/{name}', f'/usr/sbin/{name}', f'/usr/local/nginx/sbin/{name}']
    for candidate in candidates:
        if candidate and Path(candidate).is_file() and os.access(candidate, os.X_OK):
            return str(Path(candidate).resolve())
    raise DeployError(f'找不到已安装的 {name}' + ('，请用 --nginx 指定程序路径' if name == 'nginx' else ''))


def nginx_action(settings, *arguments):
    command = [settings['nginx'], *arguments]
    if settings.get('nginx_config'):
        command += ['-c', settings['nginx_config']]
    run(*command)


def preflight():
    require(os.geteuid() == 0, '请通过 sudo 执行安装或更新')
    require(platform.system() == 'Linux' and Path('/run/systemd/system').is_dir(), '需要使用 systemd 的 Linux 主机')
    require(platform.machine() in ('x86_64', 'aarch64', 'arm64'), '仅支持 Linux amd64/arm64')
    for name in ('systemctl', 'runuser'):
        require(shutil.which(name), f'找不到已安装的 {name}')
    managed_dir(STATE.parent, 0o700)
    managed_dir(ROOT)
    managed_dir(ROOT / 'releases')
    managed_dir(BACKUPS, 0o700)


def settings_from_args(args):
    host = lan_host(args.host)
    require(args.user and re.fullmatch('[a-z_][a-z0-9_-]*[$]?', args.user), '服务用户名格式不支持')
    account = pwd.getpwnam(args.user)
    require(account.pw_uid != 0 and account.pw_shell not in ('/usr/sbin/nologin', '/bin/false')
            and Path(account.pw_shell).is_file() and os.access(account.pw_shell, os.X_OK), '必须选择已有的非 root 开发账号及有效登录 shell')
    home = write_path(account.pw_dir)
    require(REPO.fullmatch(args.repo), '仓库应为 OWNER/REPO')
    paths = list(dict.fromkeys([home] + [write_path(path) for path in args.write_path]))
    require(not args.nginx_config or Path(args.nginx_config).is_absolute(), 'Nginx 主配置需要绝对路径')
    nginx_config = str(Path(args.nginx_config).resolve()) if args.nginx_config else None
    require(not nginx_config or Path(nginx_config).is_file(), '指定的 Nginx 主配置不存在')
    return {'format': 1, 'user': args.user, 'uid': account.pw_uid, 'gid': account.pw_gid,
            'origin': 'http://' + host, 'write_paths': paths, 'repo': args.repo, 'token_file': args.token_file,
            'nginx': find_program('nginx', args.nginx), 'nginx_config': nginx_config,
            'tools': {name: find_program(name) for name in ('tmux', 'rg', 'git')}}


def prepare_install(settings):
    managed_files = [CONFIG, Path('/etc/persistty/tmux.conf'), Path('/etc/systemd/system/persistty.service'),
                     Path('/etc/systemd/system/persistty-tmux.service'),
                     ROOT / 'nginx.conf',
                     Path('/usr/local/sbin/persistty-deploy')]
    require(all(not path.exists() and not path.is_symlink() for path in managed_files) and
            not (ROOT / 'current').exists() and not (ROOT / 'current').is_symlink(),
            '发现未由本安装器管理的同名配置/服务，请先人工审查迁移；不覆盖')
    require(not (DATA / 'metadata.db').exists() and not (DATA / 'tmux.sock').exists(),
            '发现已有数据库或 tmux socket，请先人工审查迁移')
    for name in ('persistty.service', 'persistty-tmux.service'):
        require(run('systemctl', 'show', '--property=LoadState', '--value', name, capture=True) == 'not-found',
                '发现已有 systemd 服务，请先人工审查迁移：' + name)
    managed_dir(CONFIG.parent, 0o700, settings['uid'], settings['gid'])
    managed_dir(DATA, 0o700, settings['uid'], settings['gid'])
    managed_dir(DATA / 'staging', 0o700, settings['uid'], settings['gid'])
    atomic_write(STATE, json.dumps(settings, indent=2) + '\n')


def render_config(package, settings, password_hash):
    template = (package / 'deploy/config.example.yaml').read_text()
    host = host_from_settings(settings)
    template = template.replace('LAN_IP', host)
    for name, program in settings['tools'].items():
        template = template.replace('/usr/bin/' + name, json.dumps(program))
    return template.replace('REPLACE_WITH_PERSISTTY_PASSWORD_OUTPUT', json.dumps(password_hash))


def render_unit(package, settings, name):
    unit = (package / f'deploy/systemd/{name}.service').read_text()
    unit = unit.replace('User=developer', 'User=' + settings['user']).replace('Group=developer', 'Group=' + str(settings['gid']))
    if name == 'persistty-tmux':
        unit = unit.replace('/usr/bin/tmux', unit_path(settings['tools']['tmux']))
    if name == 'persistty':
        unit = unit.replace('/home/developer', ' '.join(unit_path(path) for path in settings['write_paths']))
    return unit


def render_nginx(package, settings):
    return (package / 'deploy/nginx/persistty.conf').read_text().replace('LAN_IP', host_from_settings(settings))


def provision(package, settings):
    """只在首次安装（或首次安装失败的重试）执行；普通更新不改长期配置。"""
    uid, gid = settings['uid'], settings['gid']
    if not CONFIG.exists():
        try:
            with open('/dev/tty', 'rb') as tty:
                result = subprocess.run(['runuser', '-u', settings['user'], '--', str(package / 'bin/persistty'), 'password'],
                                        stdin=tty, stdout=subprocess.PIPE, check=False)
        except OSError:
            raise DeployError('首次安装需要交互终端，请下载脚本后运行，不要管道执行') from None
        password_hash = result.stdout.decode().strip()
        require(result.returncode == 0 and re.fullmatch(r'\$argon2id\$[A-Za-z0-9$=,+/]+', password_hash), '密码生成失败')
        template = render_config(package, settings, password_hash)
        atomic_write(CONFIG, template, uid=uid, gid=gid)
    private_file(CONFIG, uid)
    if not (CONFIG.parent / 'tmux.conf').exists():
        atomic_write(CONFIG.parent / 'tmux.conf', (package / 'deploy/tmux.example.conf').read_bytes(), uid=uid, gid=gid)
    private_file(CONFIG.parent / 'tmux.conf', uid)
    for name in ('persistty', 'persistty-tmux'):
        unit = render_unit(package, settings, name)
        atomic_write(Path(f'/etc/systemd/system/{name}.service'), unit, 0o644)
    atomic_write(ROOT / 'nginx.conf', render_nginx(package, settings), 0o644)
    # Nginx 由用户原有安装管理；只使用其原程序检验并 reload，不操作 nginx.service。
    nginx_action(settings, '-t')
    run('systemctl', 'daemon-reload')
    run('systemctl', 'enable', '--now', 'persistty-tmux.service')
    run('systemctl', 'enable', 'persistty.service')


def install_deployer(source):
    # 记录本次实际 Python 路径，使 sudo 的 PATH 不影响下次一键更新。
    python = str(Path(sys.executable).resolve())
    require(not any(char.isspace() for char in python), 'Python 程序路径不能包含空白')
    content = source.read_text().split('\n', 1)[1]
    atomic_write(Path('/usr/local/sbin/persistty-deploy'), '#!' + python + '\n' + content, 0o755)


def main(argv=None):
    parser = argparse.ArgumentParser(description='从 GitHub Release 一键安装或更新 Persistty（Debian）')
    commands = parser.add_subparsers(dest='command', required=True)
    install = commands.add_parser('install', help='首次安装到局域网主机')
    install.add_argument('--host', required=True, help='本机局域网 IPv4 地址；只在首次安装时指定')
    install.add_argument('--user', default=os.environ.get('SUDO_USER'), help='默认使用 sudo 前的账号')
    install.add_argument('--nginx', help='已安装的 Nginx 程序路径；通常自动找到')
    install.add_argument('--nginx-config', help='Nginx 使用非默认主配置时指定绝对路径')
    install.add_argument('--write-path', action='append', default=[])
    install.add_argument('--repo', default='coolerks/Persistty')
    update = commands.add_parser('update', help='保留配置，下载最新或指定版本并部署')
    for command in (install, update):
        command.add_argument('--version', default='latest')
        command.add_argument('--token-file', help='root 拥有的 0600 token 文件绝对路径；不传 token 本身')
    args = parser.parse_args(argv)
    preflight()
    with deployment_lock():
        if args.command == 'install':
            require(not STATE.exists() and not STATE.is_symlink(), '已有安装记录，请使用 update；首次失败也用 update 重试')
            settings = settings_from_args(args)
        else:
            require(STATE.exists(), '尚未安装，请先使用 install')
            settings = json.loads(private_file(STATE))
            require(settings.get('format') == 1 and 'nginx' in settings,
                    '部署记录不是本局域网方案，请保留旧配置并人工核对')
            host_from_settings(settings)
            account = pwd.getpwnam(settings['user'])
            require(account.pw_uid == settings['uid'] and account.pw_gid == settings['gid'] and account.pw_uid != 0,
                    '服务账号 UID/GID 发生变化，请人工检查权限')
            if args.token_file:
                settings['token_file'] = args.token_file
        arch = 'amd64' if platform.machine() == 'x86_64' else 'arm64'
        with tempfile.TemporaryDirectory(prefix='.download-', dir=ROOT) as work:
            # runuser 生成 hash 需能遍历并读取候选二进制；包目录中没有秘密。
            directory = Path(work)
            directory.chmod(0o755)
            package, manifest, digest = GitHub(settings['repo'], settings.get('token_file')).download(args.version, arch, directory)
            if args.command == 'install':
                prepare_install(settings)
            if not settings.get('provisioned'):
                provision(package, settings)
                settings['provisioned'] = True
                atomic_write(STATE, json.dumps(settings, indent=2) + '\n')
            else:
                private_file(CONFIG, settings['uid'])
                private_file(CONFIG.parent / 'tmux.conf', settings['uid'])
                nginx_action(settings, '-t')
            nginx_action(settings, '-s', 'reload')
            target = Deployment(settings).activate(package, manifest, digest)
            settings['version'] = manifest['version']
            atomic_write(STATE, json.dumps(settings, indent=2) + '\n')
            install_deployer(target / 'deploy/persistty-deploy.py')


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print('部署已中断；检查 current 与服务状态后用 update 重试。', file=sys.stderr)
        sys.exit(130)
    except (DeployError, OSError, ValueError, KeyError, tarfile.TarError, sqlite3.Error) as error:
        # 网络错误在下载层脱敏，不显示带 token/签名参数的 URL。
        print('部署失败：' + str(error), file=sys.stderr)
        sys.exit(1)
