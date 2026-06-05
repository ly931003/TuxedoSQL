# TuxedoSQL — 数据库可视化管理工具

## Problem
后端开发者需要一个数据库可视化管理工具。Navicat 是事实标准但收费昂贵，其 Lite 免费版连基础的表单视图都锁在付费墙后面。DBeaver 等免费替代品操作习惯与 Navicat 差异大，学习迁移成本高。市面上缺少一个**操作习惯对标 Navicat、核心功能免费开源**的替代方案。

## Evidence
- 用户亲身验证：Navicat Lite 版基础功能（如表视图）需付费解锁
- 现有免费工具（DBeaver）交互模式与 Navicat 差异显著，用户试用后放弃
- 社区中长期存在对 Navicat 开源替代的诉求，但尚无成熟方案

## Users
- **Primary**: 后端开发者，日常需要连接 MySQL 数据库进行库表查看、连接管理，习惯 Navicat 的交互模式
- **Not for**: 非技术用户（如产品经理、数据分析师）；需要高级 DBA 功能（性能剖析、备份恢复）的用户 — 这些由后续版本覆盖

## Hypothesis
我们认为 **Navicat 风格的开源数据库连接管理 + 库表展示** 将为 **后端开发者** 解决 **数据库可视化管理工具的付费锁定与操作习惯不兼容** 问题。
当我们看到 **用户能用 TuxedoSQL 完成日常的数据库连接管理和库表浏览，不再需要打开 Navicat** 时，说明我们做对了。

## Success Metrics
| Metric | Target | How measured |
|---|---|---|
| 连接管理闭环 | 用户可完成连接 CRUD + 测试连接 | 手动功能验证 |
| 库表浏览 | 展开连接后可见 databases → tables 树形结构 | 手动功能验证 |
| Navicat 习惯兼容 | 界面布局与交互模式参考 Navicat（左侧连接树 + 右侧内容区） | 主观评估 |

## Scope
**MVP — 连接管理 + 库表浏览（第一期）**

- MySQL 数据库连接支持
- 连接的创建、编辑、删除
- 连接测试（验证连接参数有效性）
- 连接分组（文件夹式管理）
- 连接列表/树形展示（左侧导航面板）
- 展开连接后浏览 databases → tables 层级
- 连接信息明文存储于本地配置文件
- 界面布局参考 Navicat（左侧连接树 + 右侧主内容区）

**Out of scope**
- SQL 编辑器 / 查询执行 — 后续阶段实现
- 数据浏览与编辑（表格视图） — 后续阶段实现
- 表结构设计 / DDL 操作 — 后续阶段实现
- 数据导入导出 — 后续阶段实现
- SSH 隧道 / SSL 连接 — 后续阶段实现
- 连接信息加密 — 后续阶段实现
- 除 MySQL 外的其他数据库 — 后续阶段逐步支持
- 多用户 / 团队协作 — 不做，本机桌面工具定位

## Delivery Milestones

| # | Milestone | Outcome | Status | Plan |
|---|---|---|---|---|
| 1 | 连接管理 MVP | 用户可创建/编辑/删除/测试 MySQL 连接，分组管理，库表树形浏览 | complete | [plan](../plans/database-connection-management.plan.md) |

## Open Questions
- [ ] 连接信息存储格式？（JSON？YAML？SQLite？） — 需在 /plan 阶段决定
- [ ] 是否需要在 MVP 就支持多标签页？ — 建议后续，MVP 单窗口即可

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| MySQL 驱动与 Go 生态兼容性问题 | 低 | 中 | go-sql-driver/mysql 是成熟方案 |
| Wails v3 尚在 beta，API 可能有变动 | 中 | 中 | 锁定当前版本，升级时评估 |
| Navicat 功能庞大，用户对「替代」期望过高 | 中 | 低 | 明确 MVP 范围，分阶段交付 |

---
*Status: DRAFT — requirements only. Implementation planning pending via /plan.*
