"""部署器的隔离回归：不调用真实 systemd/nginx，不修改系统配置。"""
import hashlib
import contextlib
import importlib.util
import io
import json
import os
import sqlite3
import stat
import struct
import tarfile
import tempfile
import types
import unittest
import urllib.error
import urllib.request
from pathlib import Path
from unittest.mock import patch

REPO = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('deployment', REPO / 'deploy/persistty-deploy.py')
d = importlib.util.module_from_spec(spec)
spec.loader.exec_module(d)
CHECKSUM = 'a' * 64
TEST_ORIGIN = 'http://10.23.45.67'


def manifest(version='build-2', arch='amd64', migrations=None):
    return {'format': 1, 'version': version, 'os': 'linux', 'arch': arch, 'commit': 'b' * 40,
            'migrations': migrations or [{'version': 1, 'checksum': CHECKSUM}]}


def package(root, version='build-2', arch='amd64'):
    root.mkdir()
    (root / 'bin').mkdir()
    header = bytearray(64)
    header[:6] = b'\x7fELF\x02\x01'
    struct.pack_into('<H', header, 18, {'amd64': 62, 'arm64': 183}[arch])
    (root / 'bin/persistty').write_bytes(header)
    for name in ('web/index.html', 'web/third-party-licenses.txt', 'LICENSE', 'backend-third-party-licenses.txt',
                 'deploy/config.example.yaml', 'deploy/persistty-deploy.py', 'deploy/tmux.example.conf',
                 'deploy/systemd/persistty.service', 'deploy/systemd/persistty-tmux.service',
                 'deploy/nginx/persistty.conf'):
        target = root / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text('<html>fixture</html>' if name == 'web/index.html' else 'fixture')
    (root / 'version.json').write_text(json.dumps(manifest(version, arch)))
    return root


@contextlib.contextmanager
def connect(path):
    with contextlib.closing(sqlite3.connect(path)) as connection:
        with connection:
            yield connection


def database(path, version=1, checksum=CHECKSUM):
    conn = sqlite3.connect(path)
    conn.execute('CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT)')
    conn.execute('INSERT INTO schema_migrations VALUES(?,?)', (version, checksum))
    conn.commit()
    conn.close()


class ValidationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name).resolve()
        self.addCleanup(self.temp.cleanup)

    def test_checksum_duplicate_and_traversal_rejected(self):
        for text in (CHECKSUM + '  a\n' + CHECKSUM + '  a\n', CHECKSUM + '  ../a\n', 'bad a'):
            with self.subTest(text=text), self.assertRaises(d.DeployError):
                d.parse_checksums(text)
        self.assertEqual(d.parse_checksums(CHECKSUM + '  a.tar.gz\n'), {'a.tar.gz': CHECKSUM})

    def test_private_file_rejects_link_and_loose_mode(self):
        target = self.root / 'secret'
        target.write_text('private')
        target.chmod(0o600)
        self.assertEqual(d.private_file(target, os.getuid()), b'private')
        link = self.root / 'link'
        link.symlink_to(target)
        with self.assertRaises(d.DeployError):
            d.private_file(link, os.getuid())
        os.link(target, self.root / 'hardlink')
        with self.assertRaises(d.DeployError):
            d.private_file(target, os.getuid())
        (self.root / 'hardlink').unlink()
        target.chmod(0o644)
        with self.assertRaises(d.DeployError):
            d.private_file(target, os.getuid())

    def test_redirect_strips_token_cross_host_and_rejects_http(self):
        req = urllib.request.Request('https://api.github.com/a', headers={'Authorization': 'Bearer secret'})
        redirect = d.Redirects().redirect_request(req, None, 302, '', {}, 'https://release-assets.githubusercontent.com/a')
        self.assertIsNone(redirect.get_header('Authorization'))
        same = d.Redirects().redirect_request(req, None, 302, '', {}, 'https://api.github.com/b')
        self.assertEqual(same.get_header('Authorization'), 'Bearer secret')
        for url in ('http://github.com/a', 'https://evil.test/a', 'https://github.com@evil.test/a', 'https://github.com:444/a'):
            with self.subTest(url=url), self.assertRaises(d.DeployError):
                d.github_url(url)

    def test_systemd_paths_escape_specifiers_and_quotes(self):
        self.assertEqual(d.unit_path('/srv/a%b"c'), '"/srv/a%%b\\"c"')
        (self.root / 'project').mkdir()
        self.assertEqual(d.write_path(str(self.root / 'project')), str(self.root / 'project'))
        (self.root / 'link').symlink_to(self.root / 'project')
        with self.assertRaises(d.DeployError):
            d.write_path(str(self.root / 'link'))

    def archive(self, members):
        archive = self.root / 'test.tar.gz'
        with tarfile.open(archive, 'w:gz') as output:
            for name, kind, data in members:
                info = tarfile.TarInfo(name)
                info.type = kind
                info.linkname = '/etc/passwd' if kind != tarfile.REGTYPE else ''
                info.size = len(data) if kind == tarfile.REGTYPE else 0
                info.mode = 0o7777
                output.addfile(info, io.BytesIO(data) if kind == tarfile.REGTYPE else None)
        destination = self.root / 'extracted'
        destination.mkdir(exist_ok=True)
        return archive, destination

    def test_safe_extract_rejects_escape_links_special_duplicate(self):
        cases = [[(name, tarfile.REGTYPE, b'x')] for name in ('/etc/passwd', 'persistty/../bad', '../bad',
                  'other/file', 'persistty//file', 'persistty/./file', 'persistty/a\\b')]
        cases += [[('persistty/link', kind, b'')] for kind in (tarfile.SYMTYPE, tarfile.LNKTYPE, tarfile.FIFOTYPE, tarfile.CHRTYPE)]
        cases += [[('persistty/a', tarfile.REGTYPE, b'1'), ('persistty/a', tarfile.REGTYPE, b'2')]]
        for members in cases:
            with self.subTest(members=members), self.assertRaises(d.DeployError):
                d.safe_extract(*self.archive(members))

    def test_extract_removes_suid_and_only_binary_executable(self):
        archive, destination = self.archive([('persistty', tarfile.DIRTYPE, b''),
            ('persistty/bin/persistty', tarfile.REGTYPE, b'bin'), ('persistty/web/index.html', tarfile.REGTYPE, b'html')])
        d.safe_extract(archive, destination)
        self.assertEqual(stat.S_IMODE((destination / 'persistty/bin/persistty').stat().st_mode), 0o755)
        self.assertEqual(stat.S_IMODE((destination / 'persistty/web/index.html').stat().st_mode), 0o644)

    def test_archive_size_and_file_count_bounded(self):
        archive, destination = self.archive([('persistty/a', tarfile.REGTYPE, b'12345')])
        with patch.object(d, 'MAX_EXPANDED', 4), self.assertRaises(d.DeployError):
            d.safe_extract(archive, destination)
        archive, destination = self.archive([('persistty/a', tarfile.DIRTYPE, b''),
                                            ('persistty/b', tarfile.DIRTYPE, b'')])
        with patch.object(d, 'MAX_FILES', 1), self.assertRaises(d.DeployError):
            d.safe_extract(archive, destination)

    def test_package_checks_version_arch_binary_migration_and_required_files(self):
        candidate = package(self.root / 'package')
        self.assertEqual(d.validate_package(candidate, 'build-2', 'amd64')['version'], 'build-2')
        with self.assertRaises(d.DeployError):
            d.validate_package(candidate, 'other', 'amd64')
        with self.assertRaises(d.DeployError):
            d.validate_package(candidate, 'build-2', 'arm64')
        (candidate / 'bin/persistty').write_bytes(b'#!/bin/sh\n')
        with self.assertRaises(d.DeployError):
            d.validate_package(candidate, 'build-2', 'amd64')

    def test_manifest_migration_duplicates_and_missing_license_rejected(self):
        candidate = package(self.root / 'package')
        info = manifest(migrations=[{'version': 1, 'checksum': CHECKSUM}] * 2)
        (candidate / 'version.json').write_text(json.dumps(info))
        with self.assertRaises(d.DeployError):
            d.validate_package(candidate, 'build-2', 'amd64')
        (candidate / 'version.json').write_text(json.dumps(manifest()))
        (candidate / 'web/third-party-licenses.txt').unlink()
        with self.assertRaises(d.DeployError):
            d.validate_package(candidate, 'build-2', 'amd64')

    def test_backup_includes_committed_wal_and_checks_integrity(self):
        source = self.root / 'metadata.db'
        connection = sqlite3.connect(source)
        self.addCleanup(connection.close)
        connection.execute('PRAGMA journal_mode=WAL')
        connection.execute('CREATE TABLE data(value TEXT)')
        connection.execute("INSERT INTO data VALUES('from-wal')")
        connection.commit()
        self.assertTrue(Path(str(source) + '-wal').exists())
        output = self.root / 'backup.db'
        self.assertTrue(d.backup_database(source, output))
        with connect(output) as conn:
            self.assertEqual(conn.execute('SELECT value FROM data').fetchone()[0], 'from-wal')
        self.assertEqual(stat.S_IMODE(output.stat().st_mode), 0o600)
        with self.assertRaises(d.DeployError):
            d.backup_database(source, output)

    def test_db_compatibility_rejects_newer_and_changed_checksum(self):
        source = self.root / 'metadata.db'
        database(source)
        self.assertTrue(d.compatible_database(source, manifest()))
        with connect(source) as conn:
            conn.execute('UPDATE schema_migrations SET checksum="changed"')
        self.assertFalse(d.compatible_database(source, manifest()))
        with connect(source) as conn:
            conn.execute('UPDATE schema_migrations SET version=2, checksum=?', (CHECKSUM,))
        self.assertFalse(d.compatible_database(source, manifest()))

    def test_lock_rejects_second_holder_and_symlink(self):
        path = self.root / 'lock'
        fake_info = types.SimpleNamespace(st_mode=stat.S_IFREG | 0o600, st_uid=0, st_nlink=1)
        with patch.object(d.os, 'fstat', return_value=fake_info), d.deployment_lock(path):
            with self.assertRaises(d.DeployError):
                with d.deployment_lock(path):
                    pass
        path.unlink()
        path.symlink_to(self.root / 'target')
        with self.assertRaises(OSError):
            with d.deployment_lock(path):
                pass

    def test_health_checks_api_shape_not_only_http_status(self):
        url = 'http://127.0.0.1/api/v1/auth/session'
        for body, expected in ((b'{"error":{"code":"unauthenticated"}}', True), (b'html', False),
                                (b'{"error":{"code":"wrong"}}', False)):
            error = urllib.error.HTTPError(url, 401, '', {}, io.BytesIO(body))
            with patch.object(d.urllib.request, 'urlopen', side_effect=error):
                self.assertEqual(d.api_healthy(url), expected)


class DownloadTests(unittest.TestCase):
    def test_release_assets_are_pinned_to_one_response_and_checksum_before_extract(self):
        with tempfile.TemporaryDirectory() as work:
            root = Path(work)
            candidate = package(root / 'persistty')
            tar_path = root / 'fixture.tar.gz'
            with tarfile.open(tar_path, 'w:gz') as archive:
                archive.add(candidate, arcname='persistty')
            content = tar_path.read_bytes()
            assets_base = 'https://api.github.com/repos/owner/repo/releases/assets/'
            release = {'tag_name': 'build-2', 'draft': False, 'prerelease': False, 'assets': [
                {'id': 1, 'name': 'persistty-linux-amd64.tar.gz', 'url': assets_base + '1'},
                {'id': 2, 'name': 'SHA256SUMS', 'url': assets_base + '2'}]}
            github = d.GitHub('owner/repo')
            fetched = []

            def fetch(url, limit, destination=None, asset=False):
                fetched.append(url)
                if url.endswith('/latest'):
                    return json.dumps(release).encode()
                if url.endswith('/2'):
                    return (hashlib.sha256(content).hexdigest() + '  persistty-linux-amd64.tar.gz\n').encode()
                destination.write(content)
                return b''

            github.fetch = fetch
            output = root / 'download'
            output.mkdir()
            _, metadata, _ = github.download('latest', 'amd64', output)
            self.assertEqual(metadata['version'], 'build-2')
            self.assertEqual(fetched, ['https://api.github.com/repos/owner/repo/releases/latest', assets_base + '2', assets_base + '1'])
            output2 = root / 'bad-download'
            output2.mkdir()
            original = github.fetch
            github.fetch = lambda url, *args, **kwargs: (CHECKSUM + '  persistty-linux-amd64.tar.gz\n').encode() if url.endswith('/2') else original(url, *args, **kwargs)
            with self.assertRaisesRegex(d.DeployError, 'SHA-256'), patch.object(d, 'safe_extract') as extract:
                github.download('latest', 'amd64', output2)
            extract.assert_not_called()

    def test_cross_repository_asset_and_duplicate_asset_rejected(self):
        github = d.GitHub('owner/repo')
        for assets in ([{'id': 1, 'name': 'persistty-linux-amd64.tar.gz', 'url': 'https://api.github.com/repos/evil/repo/releases/assets/1'}],
                        [{'id': 1, 'name': 'SHA256SUMS', 'url': 'https://api.github.com/repos/owner/repo/releases/assets/1'}] * 2):
            github.fetch = lambda *args, **kwargs: json.dumps({'tag_name': 'build-2', 'assets': assets}).encode()
            with tempfile.TemporaryDirectory() as work, self.assertRaises(d.DeployError):
                github.download('latest', 'amd64', Path(work))


class ActivationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        base = Path(self.temp.name).resolve()
        self.root = base / 'app'
        self.root.mkdir()
        (self.root / 'releases').mkdir()
        self.data, self.backups = base / 'data', base / 'backups'
        self.data.mkdir()
        self.backups.mkdir()
        self.old = package(self.root / 'releases/build-1', version='build-1')
        d.atomic_write(self.old / '.receipt.json', json.dumps({'sha256': 'old', 'repo': 'owner/repo'}), 0o644, os.getuid(), os.getgid())
        (self.root / 'current').symlink_to(self.old)
        self.new = package(base / 'candidate')
        database(self.data / 'metadata.db')
        self.config = base / 'config.yaml'
        self.config.write_text('KEEP config and password')
        self.deployment = d.Deployment({'repo': 'owner/repo', 'origin': TEST_ORIGIN}, self.root, self.data, self.backups)
        self.calls = []
        self.command_patch = patch.object(d, 'run', side_effect=lambda *args, **kwargs: self.calls.append(args))
        self.command_patch.start()
        self.addCleanup(self.command_patch.stop)
        # Test runs unprivileged; file ownership stays with test UID.
        original_write = d.atomic_write
        self.write_patch = patch.object(d, 'atomic_write', side_effect=lambda path, data, mode=0o600, **kwargs:
                                        original_write(path, data, mode, os.getuid(), os.getgid()))
        self.write_patch.start()
        self.addCleanup(self.write_patch.stop)
        self.addCleanup(lambda: self.assertEqual(self.config.read_text(), 'KEEP config and password'))

    def assert_web_only(self):
        self.assertTrue(all(call[-1] == 'persistty.service' for call in self.calls))
        self.assertFalse(any('tmux' in arg for call in self.calls for arg in call))

    def test_success_switches_backend_and_frontend_together_and_retains_backup(self):
        with patch.object(self.deployment, 'healthy', return_value=False), patch.object(self.deployment, 'wait_healthy'), patch.object(self.deployment, 'proxy_healthy'):
            target = self.deployment.activate(self.new, manifest(), 'new')
        self.assertEqual((self.root / 'current').resolve(), target)
        self.assertTrue((target / 'bin/persistty').exists())
        self.assertTrue((target / 'web/index.html').exists())
        self.assertEqual(len(list(self.backups.glob('*.db'))), 1)
        self.assertEqual(self.calls, [('systemctl', 'stop', 'persistty.service'), ('systemctl', 'start', 'persistty.service')])
        self.assert_web_only()

    def test_same_version_healthy_is_no_restart_and_changed_digest_refused(self):
        with patch.object(self.deployment, 'healthy', return_value=True), patch.object(self.deployment, 'proxy_healthy'):
            self.deployment.activate(self.old, manifest('build-1'), 'old')
            self.assertEqual(self.calls, [])
            with self.assertRaises(d.DeployError):
                self.deployment.activate(self.old, manifest('build-1'), 'different')
        self.assertEqual(self.calls, [])

    def test_unhealthy_same_version_retries(self):
        with patch.object(self.deployment, 'healthy', return_value=False), patch.object(self.deployment, 'wait_healthy'), patch.object(self.deployment, 'proxy_healthy'):
            self.deployment.activate(self.old, manifest('build-1'), 'old')
        self.assertEqual(len(self.calls), 2)
        self.assert_web_only()

    def test_failed_health_restores_compatible_old_code_without_database_restore(self):
        with patch.object(self.deployment, 'wait_healthy', side_effect=[d.DeployError('health'), None]), patch.object(self.deployment, 'proxy_healthy'):
            with self.assertRaisesRegex(d.DeployError, 'health'):
                self.deployment.activate(self.new, manifest(), 'new')
        self.assertEqual((self.root / 'current').resolve(), self.old)
        self.assertEqual(self.calls[-1], ('systemctl', 'start', 'persistty.service'))
        self.assertTrue(d.compatible_database(self.data / 'metadata.db', manifest()))
        self.assert_web_only()

    def test_migrated_database_stops_instead_of_unsafe_old_start(self):
        def fail_health():
            with connect(self.data / 'metadata.db') as conn:
                conn.execute('INSERT INTO schema_migrations VALUES(2,?)', (CHECKSUM,))
            raise d.DeployError('health')
        with patch.object(self.deployment, 'wait_healthy', side_effect=fail_health):
            with self.assertRaises(d.DeployError):
                self.deployment.activate(self.new, manifest(), 'new')
        self.assertEqual((self.root / 'current').resolve(), self.old)
        self.assertEqual(self.calls[-1], ('systemctl', 'stop', 'persistty.service'))
        with connect(self.data / 'metadata.db') as conn:
            self.assertEqual(conn.execute('SELECT max(version) FROM schema_migrations').fetchone()[0], 2)
        with connect(next(self.backups.glob('*.db'))) as conn:
            self.assertEqual(conn.execute('SELECT max(version) FROM schema_migrations').fetchone()[0], 1)
        self.assert_web_only()

    def test_backup_failure_restarts_old_without_switching(self):
        with patch.object(d, 'backup_database', side_effect=d.DeployError('backup')), patch.object(self.deployment, 'wait_healthy'):
            with self.assertRaisesRegex(d.DeployError, 'backup'):
                self.deployment.activate(self.new, manifest(), 'new')
        self.assertEqual((self.root / 'current').resolve(), self.old)
        self.assertEqual(self.calls[-1], ('systemctl', 'start', 'persistty.service'))
        self.assert_web_only()

    def test_first_install_failure_keeps_web_stopped_for_retry(self):
        (self.root / 'current').unlink()
        with patch.object(self.deployment, 'wait_healthy', side_effect=d.DeployError('health')):
            with self.assertRaises(d.DeployError):
                self.deployment.activate(self.new, manifest(), 'new')
        self.assertEqual(self.calls[-1], ('systemctl', 'stop', 'persistty.service'))
        self.assert_web_only()


class TemplateTests(unittest.TestCase):
    def test_lan_host_accepts_private_ipv4_and_rejects_config_injection(self):
        for host in ('10.23.45.67', '172.16.4.5', '192.168.10.20'):
            with self.subTest(host=host):
                self.assertEqual(d.lan_host(host), host)
                self.assertEqual(d.host_from_settings({'origin': 'http://' + host}), host)
        for host in ('', None, '8.8.8.8', '127.0.0.1', '0.0.0.0', '169.254.1.2', '::1',
                     'localhost', '10.23.45.67:80', '10.23.45.67; root /tmp;', '10.23.45.67\n', '010.0.0.1'):
            with self.subTest(host=host), self.assertRaises(d.DeployError):
                d.lan_host(host)
        for origin in ('https://10.23.45.67', 'http://10.23.45.67/', 'http://user@10.23.45.67',
                       'http://10.23.45.67:80', 'http://8.8.8.8', None):
            with self.subTest(origin=origin), self.assertRaises(d.DeployError):
                d.host_from_settings({'origin': origin})

    def test_install_address_is_shared_by_backend_and_nginx(self):
        args = types.SimpleNamespace(host='172.20.30.40', user='alice', repo='owner/repo', write_path=[],
                                     nginx=None, nginx_config=None, token_file=None)
        account = types.SimpleNamespace(pw_uid=1234, pw_gid=1234, pw_shell='/bin/sh', pw_dir='/home/alice')
        with patch.object(d.pwd, 'getpwnam', return_value=account), \
                patch.object(d, 'write_path', side_effect=lambda path: path), \
                patch.object(d, 'find_program', side_effect=lambda name, explicit=None: '/usr/local/bin/' + name):
            settings = d.settings_from_args(args)
        # JSON 持久化再读取：更新不需要再次传入地址。
        saved = json.loads(json.dumps(settings))
        self.assertEqual(saved['origin'], 'http://172.20.30.40')
        config = d.render_config(REPO, saved, '$argon2id$fixture')
        nginx = d.render_nginx(REPO, saved)
        self.assertIn('public_origin: http://172.20.30.40', config)
        self.assertIn('listen 172.20.30.40:80;', nginx)
        self.assertIn('server_name 172.20.30.40;', nginx)
        self.assertNotIn('LAN_IP', config + nginx)

    def test_lan_proxy_supports_editor_upload_and_websocket(self):
        nginx = d.render_nginx(REPO, {'origin': TEST_ORIGIN})
        self.assertIn('listen 10.23.45.67:80;', nginx)
        self.assertIn('server_name 10.23.45.67;', nginx)
        self.assertIn('Connection "upgrade"', nginx)
        self.assertIn('client_max_body_size 49m;', nginx)
        self.assertIn('client_max_body_size 5m;', nginx)
        self.assertNotIn('ssl_certificate', nginx)

    def test_rendered_install_matches_account_and_custom_tool_paths(self):
        settings = {'user': 'alice', 'gid': 1234, 'origin': TEST_ORIGIN,
                    'write_paths': ['/home/alice', '/srv/a b%'],
                    'tools': {'tmux': '/opt/tools/tmux', 'rg': '/usr/local/bin/rg', 'git': '/usr/local/bin/git'}}
        config = d.render_config(REPO, settings, '$argon2id$fixture')
        self.assertIn('public_origin: ' + TEST_ORIGIN, config)
        self.assertIn('mode: lan_http', config)
        self.assertIn('password_hash: "$argon2id$fixture"', config)
        self.assertIn('tmux_binary: "/opt/tools/tmux"', config)
        self.assertIn('binary: "/usr/local/bin/rg"', config)
        self.assertIn('enabled: false', config)
        web = d.render_unit(REPO, settings, 'persistty')
        self.assertIn('User=alice', web)
        self.assertIn('Group=1234', web)
        self.assertIn('ReadWritePaths=/var/lib/persistty "/home/alice" "/srv/a b%%"', web)
        tmux = d.render_unit(REPO, settings, 'persistty-tmux')
        self.assertIn('ExecStart="/opt/tools/tmux" -D', tmux)
        self.assertIn('proxy_set_header Host $http_host;', d.render_nginx(REPO, settings))

    def test_existing_program_paths_resolve_and_missing_explicit_binary_fails(self):
        with tempfile.TemporaryDirectory() as work:
            program = Path(work) / 'nginx'
            program.write_text('fixture')
            program.chmod(0o755)
            with patch.object(d.shutil, 'which', return_value=str(program)):
                self.assertEqual(d.find_program('nginx'), str(program.resolve()))
                with self.assertRaises(d.DeployError):
                    d.find_program('nginx', str(program) + '-missing')

    def test_custom_nginx_uses_original_config_without_systemd_nginx(self):
        settings = {'nginx': '/usr/local/nginx/sbin/nginx', 'nginx_config': '/srv/nginx/main.conf'}
        with patch.object(d, 'run') as run:
            d.nginx_action(settings, '-t')
            d.nginx_action(settings, '-s', 'reload')
        self.assertEqual([call.args for call in run.call_args_list], [
            ('/usr/local/nginx/sbin/nginx', '-t', '-c', '/srv/nginx/main.conf'),
            ('/usr/local/nginx/sbin/nginx', '-s', 'reload', '-c', '/srv/nginx/main.conf')])

    def test_installed_deployer_records_actual_python_not_sudo_path(self):
        with patch.object(d.sys, 'executable', '/opt/python/bin/python3'), patch.object(d, 'atomic_write') as write:
            d.install_deployer(REPO / 'deploy/persistty-deploy.py')
        path, content, mode = write.call_args.args
        self.assertEqual(path, Path('/usr/local/sbin/persistty-deploy'))
        self.assertTrue(content.startswith('#!/opt/python/bin/python3\n'))
        self.assertEqual(mode, 0o755)

    def test_provision_keeps_existing_config_and_only_manages_app_units(self):
        with tempfile.TemporaryDirectory() as work:
            root = Path(work)
            config = root / 'config.yaml'
            config.write_text('existing password and settings')
            (root / 'tmux.conf').write_text('existing tmux settings')
            settings = {'uid': 1234, 'gid': 1234, 'user': 'alice', 'origin': TEST_ORIGIN, 'write_paths': ['/home/alice'],
                        'tools': {'tmux': '/usr/local/bin/tmux', 'rg': '/usr/local/bin/rg', 'git': '/usr/bin/git'},
                        'nginx': '/usr/local/nginx/sbin/nginx', 'nginx_config': None}
            with patch.object(d, 'CONFIG', config), patch.object(d, 'ROOT', root), \
                    patch.object(d, 'private_file'), patch.object(d, 'atomic_write') as write, patch.object(d, 'run') as run:
                d.provision(REPO, settings)
            written = [call.args[0] for call in write.call_args_list]
            self.assertNotIn(config, written)
            self.assertNotIn(root / 'tmux.conf', written)
            self.assertIn(root / 'nginx.conf', written)
            self.assertEqual(config.read_text(), 'existing password and settings')
            self.assertFalse(any('nginx.service' in call.args for call in run.call_args_list))
            self.assertFalse(any('apt' in call.args or 'apt-get' in call.args for call in run.call_args_list))

    def test_units_do_not_link_web_stop_to_tmux(self):
        web = (REPO / 'deploy/systemd/persistty.service').read_text()
        tmux = (REPO / 'deploy/systemd/persistty-tmux.service').read_text()
        self.assertIn('NoNewPrivileges=true', web)
        self.assertIn('ProtectSystem=strict', web)
        self.assertNotIn('PartOf=', web + tmux)
        self.assertNotIn('BindsTo=', web + tmux)
        self.assertNotIn('NoNewPrivileges=', tmux)
        self.assertNotIn('ProtectSystem=', tmux)
        self.assertNotIn('ExecStop=', web)

class GoGateTests(unittest.TestCase):
    def test_workflow_keeps_go_exit_status_and_runs_diagnostics_after_failure(self):
        import subprocess
        import textwrap
        workflow = (REPO / '.github/workflows/release.yml').read_text()
        cases = [([{'Action': 'pass', 'Package': 'p', 'Test': 'TestOK'}], 0, 0),
                 ([{'Action': 'pass', 'Package': 'p', 'Test': 'TestOK'}], 7, 7),
                 ([{'Action': 'output', 'Package': 'p', 'Test': 'TestBroken', 'Output': 'visible failure detail\n'},
                   {'Action': 'fail', 'Package': 'p', 'Test': 'TestBroken'}], 1, 1),
                 ([{'Action': 'skip', 'Package': 'p', 'Test': 'TestMissingTool'}], 0, 1)]
        for step in ('Go 测试', 'Go 桥接测试'):
            block = workflow.split('- name: ' + step + '\n', 1)[1].split('      - ', 1)[0]
            script = textwrap.dedent(block.split('run: |\n', 1)[1])
            for events, go_exit, expected in cases:
                with self.subTest(step=step, go_exit=go_exit, expected=expected), tempfile.TemporaryDirectory() as work:
                    root = Path(work)
                    (root / 'scripts').mkdir()
                    (root / 'scripts/check-go-test-log.py').write_bytes((REPO / 'scripts/check-go-test-log.py').read_bytes())
                    (root / 'bin').mkdir()
                    go = root / 'bin/go'
                    go.write_text('#!/bin/sh\nprintf "%s\\n" "$GO_EVENTS"\nexit "$GO_EXIT"\n')
                    go.chmod(0o755)
                    (root / '.cache').mkdir()
                    env = dict(os.environ, PATH=str(root / 'bin') + os.pathsep + os.environ['PATH'],
                               GO_EVENTS='\n'.join(json.dumps(event) for event in events), GO_EXIT=str(go_exit))
                    result = subprocess.run(['bash', '-e', '-c', script], cwd=root, env=env,
                                            capture_output=True, text=True, check=False)
                    self.assertEqual(result.returncode, expected, result.stderr)
                    self.assertEqual(len(list((root / '.cache').glob('release-*-tests.jsonl'))), 1)
                    if go_exit == 1:
                        self.assertIn('visible failure detail', result.stderr)

    def test_failed_test_and_build_show_diagnostics_and_remain_failed(self):
        import subprocess
        cases = [
            ([{'Action': 'output', 'Package': 'p', 'Test': 'TestBroken', 'Output': 'fixture.go:42: deadline exceeded\n'},
              {'Action': 'fail', 'Package': 'p', 'Test': 'TestBroken'}], 'fixture.go:42: deadline exceeded'),
            ([{'Action': 'build-output', 'ImportPath': 'p', 'Output': 'fixture.go:3: undefined: Missing\n'},
              {'Action': 'build-fail', 'ImportPath': 'p'}], 'fixture.go:3: undefined: Missing'),
            ([{'Action': 'output', 'Package': 'p', 'Test': 'TestMissing', 'Output': 'required tmux unavailable\n'},
              {'Action': 'skip', 'Package': 'p', 'Test': 'TestMissing'}], 'required tmux unavailable')]
        for events, diagnostic in cases:
            with self.subTest(diagnostic=diagnostic), tempfile.TemporaryDirectory() as work:
                log = Path(work) / 'events.jsonl'
                # 其他测试通过不能掩盖前面的失败/required skip。
                events = events + [{'Action': 'pass', 'Package': 'q', 'Test': 'TestOK'}]
                log.write_text(''.join(json.dumps(event) + '\n' for event in events))
                result = subprocess.run([os.sys.executable, str(REPO / 'scripts/check-go-test-log.py'), str(log)],
                                        capture_output=True, text=True, check=False)
                self.assertEqual(result.returncode, 1)
                self.assertIn(diagnostic, result.stderr)

    def test_required_skip_and_empty_log_fail_optional_probe_allowed(self):
        import subprocess
        cases = [([{'Action': 'pass', 'Package': 'p', 'Test': 'TestA'}], 0),
                 ([{'Action': 'skip', 'Package': 'p', 'Test': 'TestMissingTool'},
                   {'Action': 'pass', 'Package': 'p', 'Test': 'TestA'}], 1),
                 ([], 1),
                 ([{'Action': 'pass', 'Package': 'p', 'Test': 'TestA'},
                   {'Action': 'fail', 'Package': 'p', 'Test': 'TestB'}], 1),
                 ([{'Action': 'skip', 'Package': 'persistty/internal/search', 'Test': 'TestLocalSearchPerformance'},
                   {'Action': 'pass', 'Package': 'p', 'Test': 'TestA'}], 0)]
        for events, expected in cases:
            with self.subTest(events=events), tempfile.TemporaryDirectory() as work:
                log = Path(work) / 'events.jsonl'
                log.write_text(''.join(json.dumps(event) + '\n' for event in events))
                result = subprocess.run([os.sys.executable, str(REPO / 'scripts/check-go-test-log.py'), str(log)],
                                        capture_output=True, check=False)
                self.assertEqual(result.returncode, expected)


if __name__ == '__main__':
    unittest.main()
