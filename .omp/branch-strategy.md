# TuxedoSQL 分支策略

**更新:** 2026-08-01
**适用:** 单人/小团队（≤3 人）桌面应用开发

## 总览

采用**轻量主干开发（GitHub Flow 变体）**：`master` 始终保持可发布，功能在短命分支上开发，squash 合并回主干。不采用 GitFlow（master/develop/release/hotfix 四分支对小型项目是纯负担），v1.0 发布前不引入 release 分支。

```
master · 始终可发布
  ├─ feat/<slug> ──→ squash merge ──→ 删分支
  ├─ fix/<slug>  ──→ squash merge ──→ 删分支
  └─ chore/<slug> ─→ squash merge ──→ 删分支
master ──里程碑完成──→ 打 tag vX.Y.Z
```

## 分支类型

| 分支 | 用途 | 生命周期 |
|---|---|---|
| `master` | 默认分支，任何时刻 CI 全绿、可发布 | 常驻 |
| `feat/<slug>` | 新功能（如 `feat/table-export`） | 短命，≤ 几天 |
| `fix/<slug>` | 缺陷修复 | 短命 |
| `chore/<slug>` | 依赖升级、重构、CI/工具链（如 `chore/deps/wails`） | 短命 |
| `docs/<slug>` | 文档（按需） | 短命 |
| `release/vX.Y` | 发布冻结分支 | v1.0 前不需要，届时再引入 |

## 工作流规则

1. **合并方式：squash merge**，保持 master 历史线性；提交信息用 conventional commits（`feat:`/`fix:`/`build:`/`ci:`/`chore:`/`docs:`/`refactor:`），描述中文为主
2. **分级门槛**：
   - 单文件小修复（几行）→ 可直推 master，但**必须本地先过 `task common:check`**（lint + test + typecheck 全套）
   - 新功能 / 多文件改动 / 依赖升级 → 开分支 + PR，由 CI 把关（workflow 已支持 `pull_request` 触发）
3. 分支合并后立即删除；PR 描述写清动机与验证方式（问题 → 证据 → 验证结果）
4. 单人阶段不强制 code review；有第二名协作者时再开 branch protection（master 要求 PR + CI 通过 + squash）

## 版本与发布

- 语义化预发布版本：`v0.1.0-alpha.N` 递增
- 里程碑完成（`.omp/prds/` 跟踪）→ 升版本号（`build/config.yml`）→ 从 master 直接打 tag，不建 release 分支
- 线上问题 hotfix：`fix/<slug>` → squash 进 master → 打补丁 tag（如 `v0.1.0-alpha.N+1`）

## 依赖升级节奏（wails alpha 线专项）

wails v3 为 nightly 发布，特性/修复流动很快，**不每次 alpha 都追**：

- **固定节奏：每 2 周**开一次 `chore/deps/wails` 分支，标准升级流程：
  1. `go get github.com/wailsapp/wails/v3@最新`
  2. `go mod tidy` + `go mod verify`
  3. 同步 wails3 CLI：`go install github.com/wailsapp/wails/v3/cmd/wails3@同版本`
  4. `wails3 generate bindings -clean=true -ts`，确认 bindings 零 diff（有 diff 则前端需同步适配）
  5. 验证：`go build ./...` + `go vet ./...` + `go test ./... -count=1` + 前端 `npm run typecheck` + 冒烟运行一次
  6. 查阅发布说明，把与本项目平台相关的修复写进 commit message
- **例外提前升级**：wails 修复了当前平台相关的崩溃/安全问题（如 Linux WebKit fetch crash 类）
- Go 工具链升级单独 commit，不与功能混在一起
