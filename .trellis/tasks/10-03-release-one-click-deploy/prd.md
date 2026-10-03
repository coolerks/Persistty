# W08 GitHub 打包与局域网一键部署

## 最新需求

用户要求简化为局域网 HTTP，沿用手动安装的 Nginx 和已有 Python，不配置 SSL/WireGuard 或安装 apt 软件；同时代码需要开源，源码、模板、文档和受版本管理开发记录不能包含用户实际 IP。首次部署指定局域网地址，之后仍从 GitHub 下载最新包一键更新。

## 验收

1. 首次 install 通过 --host 接收私有 IPv4，写入目标机部署记录；Nginx listen/server_name、后端 public_origin 和代理检查使用同一地址。
2. 后续 update 不需再传地址，不重写已有密码和配置；保持单命令更新。
3. 模板使用 LAN_IP 占位；测试使用虚构地址。扫描受版本管理及待提交文件，确认无用户实际 IP。
4. 沿用现有 Nginx/Python 和开发账号，仅首次 Nginx http 块添加 include；不操作 nginx.service，不覆盖原主配置。
5. 保持 lan_http 配置/密码/鉴权/Origin/CSRF边界、包校验/备份和独立 tmux 生命周期。
6. 完成部署回归与实际打包验包；目标机/GitHub未执行项如实记录。

## 边界

单主会话，不自动提交、推送、发布或真实安装。不增加端口、DNS、证书或 VPN 配置分支，不重构终端/认证；原测试失败证据保留，资料中的用户实际地址脱敏。
