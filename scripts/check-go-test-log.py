#!/usr/bin/env python3
"""发布 CI 拒绝 required Go suite 缺依赖而静默 skip。可选实机性能用例单列。"""
import json
import sys
from collections import defaultdict, deque

OPTIONAL = {('persistty/internal/gitview', 'TestLocalRepositoryPerformance'),
            ('persistty/internal/search', 'TestLocalSearchPerformance')}
passed, failed = 0, False
output = defaultdict(lambda: deque(maxlen=20))
with open(sys.argv[1], encoding='utf-8') as source:
    for number, line in enumerate(source, 1):
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            raise SystemExit(f'Go 日志第 {number} 行不是 JSON：' + line[-4096:].rstrip()) from None
        identity = (event.get('Package', event.get('ImportPath', '')), event.get('Test', ''))
        action = event.get('Action')
        if action in ('output', 'build-output'):
            output[identity].append(event.get('Output', '')[-4096:])
        if action in ('fail', 'build-fail') or action == 'skip' and identity[1] and identity not in OPTIONAL:
            failed = True
            reason = 'required Go 测试被跳过：' if action == 'skip' else 'Go 测试失败：'
            print(reason + '/'.join(identity), file=sys.stderr)
            for text in output[identity]:
                print(text, end='' if text.endswith('\n') else '\n', file=sys.stderr)
        if action == 'pass' and identity[1]:
            passed += 1
if failed:
    raise SystemExit(1)
if not passed:
    raise SystemExit('没有已通过的 Go 测试，拒绝发布')
print(f'已核对 {passed} 个 Go pass 事件；required suites 无 skip')
