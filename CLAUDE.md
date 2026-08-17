# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

全栈模板：Go 1.25（stdlib `net/http`）后端 + Vue 3 / TS / Vite / shadcn-vue 前端 + PostgreSQL（默认）/MySQL + Redis，Docker Compose 一键开发。`server/tasks` 与 `web/src/features/tasks` 是验证全链路的参考业务模块——新增业务时复制它们的组织方式。登录走 Glosc AI SSO（`server/sso` + `web/src/features/auth`）。

## 常用命令

```bash
make dev     # Docker 开发环境（Vue 热更新 + Go + Postgres + Redis），web :5173，api :8080
make db      # 仅起 Postgres + Redis（宿主机开发第一步）
make api     # 宿主机跑 API（需另开终端）
make web     # 宿主机跑 Vite dev server（需另开终端）
make test    # server: go test -race ./... ；web: npm run build（vue-tsc + vite build）
make check   # server: go vet ./... ；web: npm run typecheck
make fmt     # gofmt -w .
make up/down # 生产形态容器（单一 api 镜像，前端已内嵌，:8080）
```

- 单跑一个 Go 测试：`cd server && go test -race ./tasks/ -run TestName -v`
- 新增 shadcn-vue 组件：`cd web && npx shadcn-vue@latest add <name>`
- CI（`.github/workflows/ci.yml`）等价于 `make test` + `make check`；GitHub Release 触发 `release.yml` 构建 amd64/arm64 镜像发布到 GHCR
- Go 进程不会隐式读取 `.env`：宿主机运行 `make api` 时需由 shell/IDE 注入环境变量，且数据库地址要改成 `localhost`（compose 内是服务名）

## 架构要点

### 请求链路

浏览器始终使用相对 URL。开发时 Vite 把 `/api`、`/health` 代理到 Go（`web/vite.config.ts` 的 `VITE_API_PROXY_TARGET`）。生产不起 Nginx：`server/Dockerfile` 的 `production` target 先用 Node 阶段构建 `web/`，再把 `web/dist` 复制进 `server/webassets/dist` 用 `go:embed` 打进二进制，由 Go 自己在同一进程内直接提供静态资源并处理 SPA 路由回退（未匹配 `/api/`、`/health/` 的路径回退到 `index.html`）。因此生产只有一个 `api` 镜像/容器；`VITE_API_BASE_URL`（默认 `/api/v1`）作为 `server/Dockerfile` 的构建参数传给内部的前端构建阶段。

### 后端（`server/`，module `github.com/gloscai/template-go-vue3-docker/server`）

- 刻意采用扁平、按领域命名的包（`tasks`、`auth`、`sso`、`cache`、`config`、`database`、`health`），**不建** controller/service/repository 分层，也不要 `utils`/`common` 包。新的独立业务（users、billing…）建同级包；依赖组装只发生在 main 包（`server.go` 的 `run()`）。
- 路由用 Go 1.22+ ServeMux 方法模式（`"GET /api/v1/tasks"`），中间件在 `server.go` 中以嵌套调用组合。
- API 约定：成功响应包一层 `{"data": ...}`；错误为 `{"error":{"code","message"}}`。前端 `web/src/api/client.ts` 依赖此结构抛 `APIError`，改动需两边同步。
- 接口归消费方：每个业务包内定义自己需要的最小 `Store` 接口（见 `tasks/handler.go`），由 main 注入 SQL 实现。
- 双驱动写法见 `tasks/store.go`：postgres 用 `$N` 占位符 + `RETURNING`，mysql 用 `?` + `LastInsertId()`。新增存储代码必须同样兼容两种驱动。

### 数据库迁移（`server/database/migrations/{postgres,mysql}/`）

- SQL 文件经 embed 打包，启动时 `AUTO_MIGRATE=true` 按文件名顺序执行未应用的迁移（`schema_migrations` 表记录版本）。
- 每次 schema 变更在**两个驱动目录各加一个同序号新文件**；单文件多条语句用 `-- statement-breakpoint` 分隔；已应用的迁移不要原地修改。
- postgres 迁移在事务内执行；mysql DDL 隐式提交，失败需人工回滚。
- 切换 MySQL：`DB_DRIVER=mysql` + 对应 `DATABASE_URL`（见 `.env.example`）。

### 单点登录（`server/sso`）

- OAuth 2.0 授权码 + PKCE 的 RP 实现；端点来自 Discovery 文档（注意真实地址在 `/api/oauth/*`，`Provider` 懒加载并缓存 15 分钟，SSO 不可用不会阻塞启动）。
- `client_secret` 与 SSO 令牌只存在服务端；浏览器只拿本站签发的会话 JWT，放在 HttpOnly + SameSite=Lax Cookie 里（`auth/session.go`）。
- `state` 与 PKCE verifier 存 Redis，回调用 `GETDEL` 消费，保证一次性；`safeRedirect` 只允许站内绝对路径，改动这两处要连带跑 `sso` 包的测试。
- 用户以 UserInfo 的 `sub` 为唯一键 upsert 到 `users` 表，JWT subject 存的是本地用户 ID。
- 未配置 `SSO_CLIENT_ID` 时走 `RegisterDisabled`，登录接口返回 `sso_disabled`，`/api/v1/auth/session` 仍是 401——前端据此区分「未登录」与「未配置」。
- 需要登录的新接口用 `Handler.RequireUser` 包一层，再用 `sso.UserFrom(ctx)` 取用户。

### 前端（`web/src/`）

- `api/` 是唯一网络层（base URL、错误归一为 `APIError`、传输类型，`credentials: 'include'` 带会话 Cookie）；`features/<domain>/` 放 Pinia store 与领域组件；`views/` 只组合页面，不直接发请求。
- 消息弹窗统一走 `lib/message.ts`（封装 element-plus-message 的 `ElMessage`/`ElMessageBox`/`ElNotification`），业务代码不直接 import 该库。
- 需要登录的路由加 `meta: { requiresAuth: true }`，`router/index.ts` 的守卫会先 `ensureLoaded()` 再决定是否跳 `/login`。
- `components/ui/` 由 shadcn-vue CLI 管理，不要手改其风格；升级用 `--dry-run` / `--diff` 先看上游变化。
- 路径别名 `@` → `web/src`。

### 配置（`server/config`）

所有运行时配置来自环境变量，`Load()` 启动时校验：`JWT_SECRET` ≥ 32 字符（非 production 有开发默认值）、`DB_DRIVER` 仅允许 postgres/mysql、`JWT_TTL` 必须为正等。SSO 相关见 `loadSSO()`：`SSO_CLIENT_ID` 为空即关闭登录；一旦设置，secret 与绝对 http(s) 的 `SSO_REDIRECT_URL` 变为必填，避免半配置状态到运行期才暴露。新增配置项时在 `Load()` 中加解析和校验，并让 compose 的 environment 段与 `.env.example` 保持同步。
