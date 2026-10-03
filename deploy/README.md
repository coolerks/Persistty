# 局域网部署

首次安装时输入部署电脑的局域网 IP，之后通过 **http://你的局域网IP** 访问。地址只写入目标机配置，不写入开源代码。沿用电脑里已经安装并运行的 Nginx 和 Python，不安装系统软件。后端由 systemd 管理，监听 `127.0.0.1:8080`；Nginx 提供前端页面并代理 API 和终端 WebSocket。

## 1. GitHub 自动打包

将本次代码提交并推送到 `main`，GitHub Actions 会构建前后端并发布到 [Releases](https://github.com/coolerks/Persistty/releases)。等工作流成功、出现 Latest 版本后，再安装。服务器无需 Go 或 Node。

每次发布有两种架构的安装包、`persistty-deploy.py` 和 `SHA256SUMS`。脚本自动选择本机架构并校验下载包。工作流见仓库里的 `.github/workflows/release.yml`。

工作流失败时，展开失败步骤查看具体错误；Go 测试的完整 JSON 日志会保存在该次运行的 `go-test-logs` 附件。修改工作流后提交并推送新代码再验证，旧运行的 Re-run 仍使用旧版本工作流。

## 2. 首次配置 Nginx

打开你**现有 Nginx 的主配置文件**，在已有 `http { ... }` 内加入一行：

```nginx
http {
    # 原有配置保留，只加这一行：
    include /opt/persistty/nginx.conf;
}
```

这里展示的是加入位置，不要用它覆盖整个文件，也不要另建第二个 `http` 块。编译安装的主配置通常位于 `/usr/local/nginx/conf/nginx.conf`，以你当前实际使用的配置为准。

此时先不 reload；安装脚本会生成被 include 的文件，再执行语法检查和 reload。脚本沿用原有 Nginx 的启动方式，不创建或操作 `nginx.service`。

生成的站点关键配置如下，`LAN_IP` 会由安装器自动替换为你输入的地址，完整模板见 [Nginx 配置](nginx/persistty.conf)：

```nginx
server {
    listen LAN_IP:80;
    server_name LAN_IP;
    root /opt/persistty/current/web;
    # 页面、API 与 WebSocket 配置由脚本完整写入。
}
```

## 3. 下载脚本并首次安装

在部署电脑上，以你日常开发使用的非 root 账号执行：

```bash
mkdir -p "$HOME/persistty-install"
cd "$HOME/persistty-install"
curl -fL -o persistty-deploy.py \
  https://github.com/coolerks/Persistty/releases/latest/download/persistty-deploy.py
read -r -p "这台电脑的局域网 IPv4 地址：" LAN_IP
sudo "$(command -v python3)" persistty-deploy.py install --host "$LAN_IP"
```

使用 `command -v python3` 找到你实际安装的 Python，避免 sudo 找到另一套 Python。脚本默认使用执行 sudo 前的账号，首次要求输入两次应用登录密码（至少 12 字节，输入时隐藏）。安装完成后，局域网里的浏览器打开 **http://你输入的局域网IP** 并登录。`--host` 只在首次安装时提供，支持私有 IPv4 地址。

脚本会查找已有 Nginx（包括 `/usr/local/nginx/sbin/nginx`）。如果你的程序或主配置在其他位置，只在首次安装时指定：

```bash
sudo "$(command -v python3)" persistty-deploy.py install --host "$LAN_IP" \
  --nginx /你的安装目录/sbin/nginx \
  --nginx-config /你的配置目录/nginx.conf
```

如果当前直接登录的是 root，则加 `--user 你的开发账号`，Persistty 本身使用非 root 账号运行。需要在 home 之外编辑项目时，首次加 `--write-path /实际项目目录`，目录必须已存在且该开发账号有写权限。

Nginx、Python、tmux、git、rg 应已可用；脚本只检查它们，不安装、不替换。首次安装失败后，修正提示的问题，再执行 `sudo "$(command -v python3)" persistty-deploy.py update` 重试。

## 4. 以后更新：一条命令

代码推送后等待 GitHub Actions 发布成功，然后在这台电脑上执行：

```bash
sudo /usr/local/sbin/persistty-deploy update
```

更新器已记住局域网 IP、实际 Python 与 Nginx 路径，无需再次输入地址。它下载最新包、备份数据库、同时切换前后端、重启 Web 并检查访问结果。原密码、后端配置、项目文件和终端任务保留；同版本且正常时不会重复重启。

需要指定某个已发布版本时：

```bash
sudo /usr/local/sbin/persistty-deploy update --version 实际的Release版本名
```

## 5. 后端配置和常用检查

后端配置生成在 `/etc/persistty/config.yaml`，完整字段见 [配置模板](config.example.yaml)。网络部分如下，`LAN_IP` 在实际配置中已替换为安装时输入的地址：

```yaml
server:
  listen: 127.0.0.1:8080
  public_origin: http://LAN_IP
  mode: lan_http
  trusted_proxies: [127.0.0.1]
```

密码 hash 由安装时输入生成，配置和数据库归实际开发账号私有。修改后端配置后执行 `sudo systemctl restart persistty.service`；浏览器使用上述地址，保持与 `public_origin` 一致。

检查时将 `LAN_IP` 设为安装时输入的地址：

```bash
sudo systemctl status persistty.service persistty-tmux.service --no-pager
sudo journalctl -u persistty.service -n 50 --no-pager
curl -i "http://$LAN_IP/api/v1/auth/session"
```

未登录时最后一个命令返回 `401` 和 `unauthenticated` 是正常结果。访问失败时先确认电脑确实拥有安装时输入的 IP、Nginx 正在运行且 include 已放进实际使用的 `http` 块。8080 端口须留给后端。

程序在 `/opt/persistty/current`，数据在 `/var/lib/persistty`，升级前的数据库备份在 `/var/backups/persistty`。升级失败会尝试恢复兼容的旧程序；数据库不会自动覆盖。更新只重启 Web，**不要重启 `persistty-tmux.service`，否则终端里的任务会结束**。

本文提供文件和操作步骤，尚未在你的目标电脑实际安装或运行 GitHub 发布。
