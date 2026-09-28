# D10 非特权 helper 协议模拟

```sh
python3 -B -m unittest discover -s tests/integration/debian/helper -p 'test_*.py'
python3 -B tests/integration/debian/run_remote.py helper
```

五个本地测试仅使用自有临时普通文件与固定合成口令管道，覆盖单次 nonce、请求转向拒绝、内容/身份变化、链接替换、取消及错误口令。`protocol.py` 不执行 sudo/PAM，也不作为生产保存实现；其直接 truncate 不是项目所需原子保存，Python 内存清零不能保证所有副本抹除。远端命令只读检测三个入口是否存在，不读取系统策略或密码。
