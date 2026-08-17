# Go + Vue 3 + Docker Starter

一个适合中小型项目的全栈模板：后端采用 Go，前端采用 Vue 3 + Vite + shadcn-vue，默认接入 PostgreSQL 与 Redis，并可通过环境变量切换至 MySQL。仓库包含真实可运行的任务 CRUD，用来验证浏览器、API、缓存和数据库之间的完整链路。

## 内置能力

- Go 1.25、`net/http`、`log/slog`、优雅停机与生产级 HTTP 超时
- PostgreSQL / MySQL 双驱动、内嵌 SQL 迁移和连接池配置
- Redis 连接与存活、就绪探针
- JWT HS256 签发与验证基础模块
- Glosc AI 单点登录（OAuth 2.0 授权码 + PKCE），会话使用 HttpOnly Cookie
- element-plus-message 统一消息弹窗（提示、确认框、通知）
- Vue 3、TypeScript、Vite、Vue Router、Pinia
- shadcn-vue `nova` 风格、Tailwind CSS v4、响应式示例页面
- 开发多阶段 Docker 镜像；生产阶段将前端构建产物内嵌进 API 二进制，单一镜像部署
- Docker Compose 一键开发、数据卷持久化
- GitHub Actions 测试，以及 Release 发布单一镜像到 GHCR

## 立即开始

要求：Docker 24+ 与 Docker Compose v2+。

```bash
make dev
```

首次构建后访问：

- Web 开发服务器：<http://localhost:5173>
- API：<http://localhost:8080/api/v1/tasks>
- 存活检查：<http://localhost:8080/health/live>
- 就绪检查：<http://localhost:8080/health/ready>

`make dev` 会同时启动 Vue 热更新、Go API、PostgreSQL 和 Redis。修改 `server` 或 `web` 下的代码后重启对应容器即可；Vite 前端修改会即时生效。

若本机 `8080` 已被占用，可以运行 `API_PORT=18080 make dev`；浏览器仍访问 `5173`，仅直连 API 的端口改为 `18080`。

如果希望直接在宿主机开发：

```bash
make init
make db
make api     # 新终端
make web     # 新终端
```

宿主机运行 API 时，请将 `.env` 中的数据库地址改为 `localhost`；Go 本身不会隐式读取 `.env`，可由 shell、IDE 或进程管理器注入这些变量。

## 生产形态运行

```bash
cp .env.example .env
# 修改 .env，尤其是 JWT_SECRET 与数据库密码
make up
```

访问 <http://localhost:8080>。生产镜像只有 `api` 一个容器：Go 二进制内嵌前端静态资源，直接提供页面并处理 `/api`、`/health` 请求，未匹配路径回退到 `index.html`（SPA 路由）。

停止容器：

```bash
make down
```

## 目录结构

```text
.
├── server/
│   ├── main.go                 # 进程入口与信号处理
│   ├── server.go               # 依赖组装、HTTP 服务和优雅停机
│   ├── middleware.go           # 日志、CORS、请求 ID、panic 恢复
│   ├── auth/                   # JWT 领域能力
│   ├── cache/                  # Redis 客户端
│   ├── config/                 # 环境配置与校验
│   ├── database/               # SQL 连接、迁移执行器
│   │   └── migrations/
│   │       ├── postgres/
│   │       └── mysql/
│   ├── health/                 # live / ready 探针
│   ├── sso/                    # OAuth2/OIDC 登录、会话、users 存储
│   ├── tasks/                  # 示例业务：模型、存储、HTTP、测试
│   ├── webassets/               # 生产构建时内嵌 web/dist 的前端资源（go:embed）
│   └── Dockerfile               # development 阶段仅供 make dev 用；production 阶段先构建 web 再嵌入
├── web/
│   ├── src/
│   │   ├── api/                # 类型化 HTTP 客户端
│   │   ├── components/         # 布局与 shadcn-vue UI 源码
│   │   ├── features/           # 按业务领域组织的功能
│   │   ├── router/             # Vue Router
│   │   └── views/              # 页面入口
│   ├── components.json         # shadcn-vue 项目配置
│   └── Dockerfile               # 仅 development 阶段（Vite 热更新），生产由 server/Dockerfile 构建
├── docs/architecture.md
├── docker-compose.yml          # 生产形态
├── docker-compose.dev.yml      # 开发形态
├── .github/workflows/
│   ├── ci.yml
│   └── release.yml
├── .env.example
└── Makefile
```

后端刻意采用扁平、按业务职责命名的包，而不是预先堆叠 `controller/service/repository`。新增业务时可以复制 `tasks` 的组织方式，让模型、SQL 与 HTTP 行为保持在同一个清晰边界内。

## API 示例

| 方法     | 路径                 | 说明                                 |
| -------- | -------------------- | ------------------------------------ |
| `GET`    | `/health/live`       | 进程存活                             |
| `GET`    | `/health/ready`      | 数据库和 Redis 就绪                  |
| `GET`    | `/api/v1/tasks`      | 查询最近 100 条任务                  |
| `POST`   | `/api/v1/tasks`      | 创建任务，Body: `{"title":"..."}`    |
| `PATCH`  | `/api/v1/tasks/{id}` | 更新状态，Body: `{"completed":true}` |
| `DELETE` | `/api/v1/tasks/{id}` | 删除任务                             |
| `GET`    | `/api/v1/auth/sso/login`    | 跳转到 SSO 授权页（浏览器访问） |
| `GET`    | `/api/v1/auth/sso/callback` | SSO 回调，成功后写入会话 Cookie |
| `GET`    | `/api/v1/auth/session`      | 当前登录用户，未登录返回 401     |
| `POST`   | `/api/v1/auth/logout`       | 清除本站会话，并返回 SSO 登出地址 |

## 单点登录（Glosc AI SSO）

前端只做跳转，`client_secret` 与 SSO 令牌全部留在服务端：浏览器访问 `/api/v1/auth/sso/login` → SSO 授权页 → 回调带回一次性 `code` → 服务端换取令牌并读取 UserInfo → 以 `sub` 为主键写入本地 `users` 表 → 签发本站自己的会话 JWT，放进 HttpOnly Cookie。

1. 在 <https://sso.gloscai.com/dashboard/clients> 创建客户端，启用 `authorization_code`。
2. 回调地址必须与白名单逐字一致（协议、域名、端口、路径）：
   - 开发：`http://localhost:5173/api/v1/auth/sso/callback`（浏览器访问 5173，Vite 代理到 API）
   - 生产：`https://<你的域名>/api/v1/auth/sso/callback`
3. 在 `.env` 填入配置后重启：

```env
SSO_CLIENT_ID=<客户端 ID>
SSO_CLIENT_SECRET=<客户端密钥>
SSO_REDIRECT_URL=http://localhost:5173/api/v1/auth/sso/callback
SESSION_COOKIE_SECURE=false   # 仅 HTTPS 下改为 true，生产默认 true
```

访问 <http://localhost:5173/login> 登录，成功后跳转 `/profile`。

留空 `SSO_CLIENT_ID` 即关闭登录：`/api/v1/auth/session` 仍返回 401，登录接口返回 `sso_disabled`，模板其余部分照常运行。端点地址来自 SSO 的 Discovery 文档（`SSO_ISSUER` + `/api/.well-known/openid-configuration`），首次使用时读取并缓存 15 分钟，因此 SSO 暂时不可用不会阻塞 API 启动。

安全要点：`state` 与 PKCE `code_verifier` 存在 Redis 中且只能消费一次（`GETDEL`），登录回调的 `redirect_to` 只接受站内绝对路径，防止开放重定向。

## 消息弹窗

`web/src/lib/message.ts` 是统一入口，封装 [element-plus-message](https://github.com/3DMXM/element-plus-message)（Element Plus 中裁剪出的 `ElMessage` / `ElMessageBox` / `ElNotification`）：

```ts
import { confirmAction, notifyError, notifySuccess } from '@/lib/message'

notifySuccess('任务已创建')
if (await confirmAction('确定删除吗？', '删除任务', { danger: true })) {
  // 用户已确认
}
```

业务代码只依赖这个模块，不直接 import 该库，方便日后整体替换弹窗实现。

## 数据库与迁移

默认使用 PostgreSQL。API 启动时在 `AUTO_MIGRATE=true` 的情况下读取对应驱动目录中的 SQL，并按文件名顺序执行尚未应用的迁移。PostgreSQL 迁移在事务中执行；MySQL 的 DDL 会隐式提交，因此失败时可能需要人工回滚已经执行的语句。

切换 MySQL：

```env
DB_DRIVER=mysql
DATABASE_URL=app:app@tcp(localhost:3306)/app?parseTime=true&charset=utf8mb4
```

为每次 schema 修改新增有序文件，例如：

```text
server/database/migrations/postgres/002_add_users.sql
server/database/migrations/mysql/002_add_users.sql
```

一个文件包含多个 SQL 语句时，用 `-- statement-breakpoint` 分隔。已投产的迁移不要原地修改。

## 常用命令

```bash
make help      # 查看命令
make dev       # 一键开发
make test      # Go race tests + Vue 类型与构建
make check     # go vet + vue-tsc
make fmt       # gofmt
make logs      # 容器日志
```

新增 shadcn-vue 组件时，在 `web` 目录运行：

```bash
npx shadcn-vue@latest docs dialog
npx shadcn-vue@latest add dialog
```

## 发布镜像

在 GitHub 创建 Release 后，`release.yml` 会构建 `linux/amd64` 与 `linux/arm64` 的单一镜像并发布：

```text
ghcr.io/<owner>/<repo>:<version>
```

仓库的 Actions 需要保持 `packages: write` 权限。发布到 GHCR 与部署到具体服务器是两个独立步骤；服务器部署可在此工作流后追加 SSH、Kubernetes、Nomad 或云平台步骤。

## 开始自己的项目

1. 在 GitHub 将本仓库标记为 Template repository，然后点击 **Use this template**。
2. 全局替换 Go module `github.com/gloscai/template-go-vue3-docker/server`。
3. 修改 `web/package.json` 名称、页面品牌、`.env.example` 与镜像名。
4. 保留 `tasks` 作为参考，或在第一个业务模块完成后删除它。
5. 首次公开部署前更换所有密码与 `JWT_SECRET`。

更多设计取舍见 [docs/architecture.md](docs/architecture.md)。


## 社区
- https://linux.do/