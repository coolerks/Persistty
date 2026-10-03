# 局域网部署设计

首次：现有 Nginx http 块 include /opt/persistty/nginx.conf → 实际 Python install --host 局域网IPv4 → 输入应用密码。后续仍为 sudo /usr/local/sbin/persistty-deploy update。

## 修改归属与边界

- 部署器负责 --host 的 RFC1918 IPv4 校验，阻止配置指令注入；state.origin 持久化为 http://地址，更新校验并读取该值。无需新增第二份地址状态或默认个人IP。
- config.example.yaml 和 nginx/persistty.conf 使用 LAN_IP 占位；render_config/render_nginx 从同一已保存 origin 生成真实目标配置。模板不直接运行。
- 测试使用虚构地址，覆盖三种私有网段、非法/公网/注入输入、持久化后生成配置；Go模式矩阵仅替换测试值，产品行为不变。
- README、规范、任务和日志去除个人 IP，不删除原失败结论。目标机生成的私有配置不进入仓库。

## 保留行为

已有 Nginx/Python 路径、非 root 开发账号、下载校验、SQLite 一致备份、原子版本切换和 tmux 独立生命周期不变。更新保留密码、后端配置、项目和终端任务；不安装依赖或扩展 SSL/VPN/端口/DNS 支持。
