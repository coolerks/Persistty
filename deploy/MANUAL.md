# Debian 手工部署 Persistty

本包包含已构建的前端、Linux 后端、后端配置模板、systemd 服务和 Nginx 站点。把包上传到 Debian 后即可部署，服务器无需 Go、Node 或下载 GitHub Release。本文所有命令在 Debian 上执行。

本次交付为 `persistty-linux-amd64.tar.gz`，对应 `uname -m` 输出 `x86_64`。包内 `version.json` 记录实际架构和构建版本。沿用已有 Nginx；访问方式为局域网 HTTP，实际 IP 仅写入你服务器的配置。

## 1. 文件放在哪里

| 包内文件 | Debian 位置 | 用途 |
| --- | --- | --- |
| `bin/persistty` | `/opt/persistty/current/bin/persistty` | 后端程序 |
| `web/` 全部内容 | `/opt/persistty/current/web/` | 前端静态文件，包含 index.html、assets 等 |
| `deploy/config.example.yaml` | `/etc/persistty/config.yaml` | 修改 IP、密码 hash、工具路径后的后端配置 |
| `deploy/tmux.example.conf` | `/etc/persistty/tmux.conf` | 独立终端 server 配置 |
| `deploy/systemd/persistty.service` | `/etc/systemd/system/persistty.service` | Web 后端服务 |
| `deploy/systemd/persistty-tmux.service` | `/etc/systemd/system/persistty-tmux.service` | 独立 tmux 服务 |
| `deploy/nginx/persistty.conf` | `/opt/persistty/nginx.conf` | 加入现有 Nginx http 块的站点配置 |
| 自动生成的数据 | `/var/lib/persistty/` | 数据库、上传暂存、tmux socket，升级保留 |

`current` 是指向 `/opt/persistty/releases/构建目录` 的软链接；前后端同时切换版本。包内 `deploy/persistty-deploy.py` 是原 GitHub 下载更新器，**本次手工部署不需要运行它**。提权编辑默认关闭，`deploy/elevation/` 无需安装。

## 2. 解压与准备

以你日常开发使用的已有非 root 账号登录 Debian。若当前为 root，先 `su - 你的开发账号`。后端和终端使用这个账号，能访问其有权限的项目目录。

把安装包及 `SHA256SUMS` 上传到同一目录，再执行：

```bash
sha256sum -c SHA256SUMS --ignore-missing
tar -xzf persistty-linux-amd64.tar.gz
cd persistty
uname -m
command -v tmux git rg
```

tmux、git、rg 须已经可用。若工具在自定义目录，稍后填写实际绝对路径；本文不安装 Nginx、Python 或其他系统软件。Nginx 须已经运行，端口80供该站点使用，后端使用 `127.0.0.1:8080`。

以下变量和后续命令在同一终端执行：

```bash
SERVICE_USER="$(id -un)"
SERVICE_GROUP="$(id -gn)"
SERVICE_HOME="$(getent passwd "$SERVICE_USER" | cut -d: -f6)"
test "$(id -u)" -ne 0 || { echo '请切换到非 root 开发账号'; exit 1; }
RELEASE_DIR="/opt/persistty/releases/$(date +%Y%m%d-%H%M%S)"
sudo install -d -m 755 "$RELEASE_DIR/bin" "$RELEASE_DIR/web"
sudo install -m 755 bin/persistty "$RELEASE_DIR/bin/persistty"
sudo cp -R web/. "$RELEASE_DIR/web/"
sudo cp version.json LICENSE backend-third-party-licenses.txt "$RELEASE_DIR/"
sudo chown -R root:root "$RELEASE_DIR"
sudo ln -sfn "$RELEASE_DIR" /opt/persistty/current
sudo install -d -m 700 -o "$SERVICE_USER" -g "$SERVICE_GROUP" /etc/persistty /var/lib/persistty
sudo install -m 600 -o "$SERVICE_USER" -g "$SERVICE_GROUP" deploy/config.example.yaml /etc/persistty/config.yaml
sudo install -m 600 -o "$SERVICE_USER" -g "$SERVICE_GROUP" deploy/tmux.example.conf /etc/persistty/tmux.conf
```

这是首次安装流程。已有部署的升级按照第7节执行，保留原配置和数据。

## 3. 填好后端配置

运行以下命令，从终端隐藏输入并确认应用登录密码（至少12字节）：

```bash
/opt/persistty/current/bin/persistty password
```

将输出的 `$argon2id$...` 完整复制到 `/etc/persistty/config.yaml` 的 `auth.password_hash`，使用**单引号**包住整段。用你已有的编辑器修改这个文件：

```yaml
server:
  listen: 127.0.0.1:8080
  public_origin: http://LAN_IP
  mode: lan_http
  trusted_proxies: [127.0.0.1]
auth:
  password_hash: '把刚生成的完整hash放在这里'
  session_ttl: 168h
```

将 `LAN_IP` 改成 Debian 电脑的局域网 IPv4 地址。完整模板的其他字段保留；核对 `terminal.tmux_binary`、`search.binary`、`git.binary` 为 `command -v` 显示的绝对路径。`terminal.shell` 默认检测该账号的登录 shell，需要指定时取消注释并填实际路径。

配置必须归服务账号所有、权限600；父目录权限700。密码及实际配置留在服务器上，勿提交到源码仓库。

## 4. 安装 systemd 服务

```bash
sudo install -m 644 deploy/systemd/persistty.service /etc/systemd/system/persistty.service
sudo install -m 644 deploy/systemd/persistty-tmux.service /etc/systemd/system/persistty-tmux.service
sudoedit /etc/systemd/system/persistty.service /etc/systemd/system/persistty-tmux.service
```

两份服务的 `User=developer` 和 `Group=developer` 分别改为上面的 `$SERVICE_USER` 和 `$SERVICE_GROUP` 实际值。保留 tmux 与 Web 两个独立服务。

Web 服务的 `ReadWritePaths=/var/lib/persistty /home/developer` 中，将 `/home/developer` 改为 `$SERVICE_HOME` 的实际值。如果项目在 home 外面，在同一行追加已有项目目录，例如 `/srv/projects`；服务账号本身也必须对目录有写权限。其余配置保留。

tmux 服务 `ExecStart` 中的 `/usr/bin/tmux` 改成已有 tmux 的实际路径，保留后续参数。配置完成后：

```bash
sudo systemd-analyze verify /etc/systemd/system/persistty.service /etc/systemd/system/persistty-tmux.service
sudo systemctl daemon-reload
sudo systemctl enable --now persistty-tmux.service
sudo systemctl enable --now persistty.service
sudo systemctl status persistty.service persistty-tmux.service --no-pager
curl -i http://127.0.0.1:8080/api/v1/auth/session
```

最后一条匿名请求返回 **401 / unauthenticated** 表示后端正常。查看失败原因：`sudo journalctl -u persistty.service -n 50 --no-pager`；tmux 服务问题使用 `-u persistty-tmux.service`。依赖和项目目录路径须真实存在。

## 5. 接入现有 Nginx

```bash
sudo install -m 644 deploy/nginx/persistty.conf /opt/persistty/nginx.conf
sudoedit /opt/persistty/nginx.conf
```

将文件开头的 `listen LAN_IP:80`、`server_name LAN_IP` 都替换成第3节使用的**同一个局域网 IPv4**，其他配置保留。该文件已包含前端路由、API、上传和终端 WebSocket 代理。

打开你正在使用的 Nginx 主配置，在其已有 `http { ... }` 内添加一行：

```nginx
include /opt/persistty/nginx.conf;
```

不要另外创建 `http` 块或覆盖主配置。编译安装的主配置常见于 `/usr/local/nginx/conf/nginx.conf`，以你的实际安装为准。用原有程序检查、重载，例如：

```bash
sudo /usr/local/nginx/sbin/nginx -t
sudo /usr/local/nginx/sbin/nginx -s reload
```

若当前 Nginx 使用自定义主配置，两条命令都追加 `-c /实际位置/nginx.conf`。这里直接使用已有 Nginx 程序，不需要配置 `nginx.service`。

## 6. 访问与检查

局域网浏览器打开 `http://你的局域网IP`，使用第3节的密码登录，添加项目文件夹并创建终端。登录、编辑保存、文件上传和终端连接都通过同一 Nginx 地址访问。

- 打不开页面：检查 Nginx 正在运行、主配置 include、IP与端口、局域网连通和主机防火墙是否允许局域网访问80端口。
- 页面可开但API返回502：查看 `persistty.service` 日志与 `127.0.0.1:8080` 的匿名检查。
- 登录或终端返回403：浏览器地址须与后端 `public_origin` 完全一致；使用IP访问，不换成其他主机名或端口。
- 终端不可用：检查独立 `persistty-tmux.service`、实际工具路径以及 `/var/lib/persistty/tmux.sock`；两个服务须用同一账号。
- 能浏览项目却无法保存：检查服务账号权限及 Web 服务 `ReadWritePaths`，修改 unit 后执行 daemon-reload，再重启 Web 服务。

若旧包在浏览器报 `crypto.randomUUID is not a function`，使用 `manual-20261003-http-uuid` 或后续包含该修复的包，按第7节升级并在浏览器强制刷新。此问题已在前端兼容普通局域网 HTTP，无需更改现有 HTTP/Nginx 配置。

## 7. 以后手工升级

上传并校验新的同架构包，解压进入新包目录。仅复制程序和前端到新版本目录，保持配置和数据：

```bash
RELEASE_DIR="/opt/persistty/releases/$(date +%Y%m%d-%H%M%S)"
sudo install -d -m 755 "$RELEASE_DIR/bin" "$RELEASE_DIR/web"
sudo install -m 755 bin/persistty "$RELEASE_DIR/bin/persistty"
sudo cp -R web/. "$RELEASE_DIR/web/"
sudo cp version.json LICENSE backend-third-party-licenses.txt "$RELEASE_DIR/"
sudo chown -R root:root "$RELEASE_DIR"
sudo systemctl stop persistty.service
BACKUP_DIR="/var/backups/persistty/$(date +%Y%m%d-%H%M%S)"
sudo install -d -m 700 "$BACKUP_DIR"
sudo tar --exclude='./tmux.sock' -C /var/lib/persistty -cpf "$BACKUP_DIR/data.tar" .
sudo chmod 600 "$BACKUP_DIR/data.tar"
sudo ln -sfn "$RELEASE_DIR" /opt/persistty/current
sudo systemctl start persistty.service
sudo systemctl status persistty.service --no-pager
```

停止 Web 后备份数据目录，数据库及其 WAL/SHM 一并保留，排除正在运行的 tmux socket；备份目录仅 root 可读。重载已有 Nginx，再检查页面、登录和终端。不要覆盖 `/etc/persistty/config.yaml` 或删除 `/var/lib/persistty`。

**升级和修改配置仅重启 `persistty.service`。停止或重启 `persistty-tmux.service` 会终止其中运行的终端任务。** 主机重启也不会恢复正在运行的终端进程。

本包在本机构建并交叉编译，目标 Debian 的 systemd/Nginx 启动需按照本文在你的电脑上实际验证。
