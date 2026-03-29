#!/bin/bash
# Epic-D 一键验收脚本
# 运行方式：bash .epic-acceptance.sh

set -e

PORT=${PORT:-8080}
BASE_URL="http://localhost:${PORT}"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $1"; ((pass_count++)); }
fail() { echo -e "${RED}[FAIL]${NC} $1"; ((fail_count++)); }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

pass_count=0
fail_count=0

echo "============================================"
echo "  Epic-D 验收 - AgentFlow 0-1 MVP"
echo "============================================"
echo ""

# 检查服务是否运行
info "检查 Go 核心服务..."
if curl -s --max-time 2 "${BASE_URL}/api/agents" > /dev/null 2>&1; then
    pass "Go 核心服务运行正常 (${BASE_URL})"
else
    fail "Go 核心服务未启动，请先运行: go run ./cmd/"
    echo ""
    echo "============================================"
    echo "  验收结果: $pass_count PASS, $fail_count FAIL"
    echo "============================================"
    exit 1
fi

# 1. Agent注册
info "1. 测试 Agent 注册..."
AGENT_RESP=$(curl -s -X POST "${BASE_URL}/api/agents" \
    -H "Content-Type: application/json" \
    -d '{"name":"test-agent","model":"gpt-4","provider":"openai"}')
if echo "$AGENT_RESP" | grep -q '"id"'; then
    pass "Agent 注册成功"
    AGENT_ID=$(echo "$AGENT_RESP" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    info "Agent ID: $AGENT_ID"
else
    fail "Agent 注册失败: $AGENT_RESP"
    echo ""
    echo "============================================"
    echo "  验收结果: $pass_count PASS, $fail_count FAIL"
    echo "============================================"
    exit 1
fi

# 2. 心跳上报
info "2. 测试心跳上报..."
HB_RESP=$(curl -s -X POST "${BASE_URL}/api/agents/${AGENT_ID}/heartbeat")
if echo "$HB_RESP" | grep -q '"status":"ok"'; then
    pass "心跳上报成功"
else
    fail "心跳上报失败: $HB_RESP"
fi

# 3. 状态上报(running)
info "3. 测试状态上报(running)..."
STATUS_RESP=$(curl -s -X POST "${BASE_URL}/api/agents/${AGENT_ID}/status" \
    -H "Content-Type: application/json" \
    -d '{"status":"running","description":"任务进行中"}')
if echo "$STATUS_RESP" | grep -q '"status":"ok"'; then
    pass "状态上报(running)成功"
else
    fail "状态上报失败: $STATUS_RESP"
fi

# 4. 状态上报(failed) - 飞书通知（需要配置，未配置则跳过）
info "4. 测试状态上报(failed)..."
STATUS_RESP=$(curl -s -X POST "${BASE_URL}/api/agents/${AGENT_ID}/status" \
    -H "Content-Type: application/json" \
    -d '{"status":"failed","description":"测试失败"}')
if echo "$STATUS_RESP" | grep -q '"status":"ok"'; then
    pass "状态上报(failed)成功"
    info "飞书通知: 请手动确认是否收到（需配置飞书 WebHook）"
else
    fail "状态上报失败: $STATUS_RESP"
fi

# 5. SSE实时页面
info "5. 检查 SSE 实时页面..."
if curl -s --max-time 2 "http://localhost:8081/internal/monitor/monitor.php" > /dev/null 2>&1; then
    pass "SSE 页面可访问 (http://localhost:8081)"
else
    info "SSE 页面未启动（如需测试请运行: php -S 0.0.0.0:8081 -t . &）"
fi

# 6. Go 测试
info "6. 运行 Go 测试..."
export PATH="$HOME/.local/go/bin:$PATH"
if go test ./... > /dev/null 2>&1; then
    pass "Go 测试全部通过"
else
    fail "Go 测试失败，请运行: go test ./... 查看详情"
fi

# 清理测试数据
info "清理测试数据..."
curl -s -X DELETE "${BASE_URL}/api/agents/${AGENT_ID}" > /dev/null 2>&1 || true
pass "清理完成"

echo ""
echo "============================================"
echo "  验收结果: $pass_count PASS, $fail_count FAIL"
echo "============================================"

if [ $fail_count -eq 0 ]; then
    echo -e "${GREEN}Epic-D 验收通过！${NC}"
    exit 0
else
    echo -e "${RED}Epic-D 验收失败，请修复上述问题${NC}"
    exit 1
fi
