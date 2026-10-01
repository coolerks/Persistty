# W05/W06 Debian 包测试

仅在用户明确授权目标与项目载荷传输后运行。连接由现有 `remote_config.py` 从私有 `.env` 安全读取，不打印值。上传的载荷是本项目 Linux 测试二进制、当前 runner 和合成共享 JSON；不能因为已编译或 `.env` 存在就推断远端授权。

```sh
mkdir -p /private/tmp/persistty-w05-w06-acceptance-bin
for pkg in files search gitview toolrunner httpapi; do
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c \
    -o /private/tmp/persistty-w05-w06-acceptance-bin/w06-$pkg.test ./internal/$pkg || exit 1
done
python3 -B tests/integration/debian/w06/run.py \
  --binaries /private/tmp/persistty-w05-w06-acceptance-bin
```

每轮仅新建0700的 `/tmp/persistty-w06-accept-*`。所有 Go临时树位于自己的 `TMPDIR`，要求目标实际 Git/rg存在；httpapi只运行W05预览/转义正文保存与W06共享DTO/HTTP会话链路，其余四包全量运行。原始测试正文不回显，报告只含版本、测试名、退出码、二进制SHA256和清理判定。任何 skip、零测试或非零退出均判失败；没有远端Go安装、系统安装或正式服务修改。

固定进程组、35秒测试期限、40秒进程期限、2MiB输出上限。正常异常后精确清理已验证ROOT；清理失败报告 `recovery_root`，保留待核查身份，不能声称零残留。runner被SIGKILL或网络断开不能保证finally执行，不使用通配清理。优化模式与非法root在执行前拒绝，本地安全回归：

```sh
python3 -B -m unittest discover -s tests/integration/debian/w06 -p 'test_run.py'
```

浏览器联调沿用[现有隔离 harness](../browser/README.md)，包测试不代替 systemd/浏览器或真实手机验收。
