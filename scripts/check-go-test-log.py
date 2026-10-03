#!/usr/bin/env python3
"""发布 CI 拒绝 required Go suite 缺依赖而静默 skip。可选实机性能用例单列。"""
import json
import sys

OPTIONAL = {('persistty/internal/gitview', 'TestLocalRepositoryPerformance'),
            ('persistty/internal/search', 'TestLocalSearchPerformance')}
passed = 0
for line in open(sys.argv[1]):
    event = json.loads(line)
    if event.get('Action') == 'fail':
        raise SystemExit('Go 测试失败：' + str(event.get('Package')) + '/' + str(event.get('Test', '')))
    if event.get('Action') == 'skip' and event.get('Test'):
        identity = (event.get('Package'), event['Test'])
        if identity not in OPTIONAL:
            raise SystemExit('required Go 测试被跳过：' + '/'.join(identity))
    if event.get('Action') == 'pass' and event.get('Test'):
        passed += 1
if not passed:
    raise SystemExit('没有已通过的 Go 测试，拒绝发布')
print(f'已核对 {passed} 个 Go pass 事件；required suites 无 skip')
