# Epic-E 技术方案：单节点Agent管理

> 状态：待老板验证 Epic-D 后执行
> 创建时间：2026-03-29

## 方案来源
- PRD：~/tmp/prd/AgentFlow 产品需求文档（PRD）- 分阶段拆分版.md
- 参考技术文档：~/tmp/prd/AgentFlow 技术文档（对应分阶段PRD）.md
- 唐僧独立设计，对比参考文档后选型

## 核心模块

### 1. 标签管理
- SQLite 新增 agent_tags 表
- 支持多维度标签（env/role/version/capacity）
- REST API：POST /api/agents/:id/tags

### 2. 任务队列
- Go 协程 + slice 实现优先级队列（高/中/低）
- FIFO 调度，高优先级优先
- 存储：SQLite 新增 tasks 表

### 3. 资源限制
- CPU/内存/并发数上限
- 超限拒绝新任务，记录日志
- 用 Go 标准库 `/proc` 读取资源

### 4. 离线恢复
- SQLite 保留 Agent 历史状态
- 重连后自动恢复在线状态
- 同步离线期间未上报的状态

### 5. 状态详情
- failed 携带原因（description 字段）
- 状态历史查询（默认7天）

## 架构取舍

| 取舍 | 决策 | 理由 |
|------|------|------|
| 不用 Redis Stream | ✅ 放弃 | 单节点 Go Channel 够用，引入 Redis 增加复杂度 |
| 不用 MySQL | ✅ 放弃 | SQLite + 抽象接口过渡，Epic-D 已用 SQLite |
| 不用 ECharts | ✅ 放弃 | 单节点阶段简单数字大盘足够 |
| 用 Viper | ✅ 采用 | 参考文档一致，配置热更新需要 |
| 保留存储抽象接口 | ✅ 采用 | 切换 MySQL 只需改配置 |

## 下游 Epic
- Epic-F（实时观测大盘）
- Epic-G（告警通知）
- Epic-H（扩展性）
