# 架构说明

## 请求链路

```text
Browser
  └─ Vue view / Pinia feature
       └─ typed API client
            └─ Nginx or Vite proxy
                 └─ Go middleware
                      └─ tasks handler + SQL store
                           └─ PostgreSQL / MySQL

Go API ── health readiness ── Redis
```

开发环境中，Vite 把 `/api` 和 `/health` 转发到 `api:8080`。生产环境由 Nginx 完成相同工作，因此浏览器始终使用相对 URL，不需要在公开镜像中写入后端主机名。

## 后端边界

- `main` 负责依赖组装和进程生命周期，不承载业务规则。
- `tasks` 是一个完整业务边界，拥有类型、数据库操作、HTTP 行为与测试。
- `database` 只负责连接和迁移，不抽象通用 Repository。
- `auth` 提供 JWT 基础能力；接入用户业务时，由消费令牌的业务包定义它需要的最小接口。
- `cache` 和 `health` 保持小而直接，避免无明确归属的 `utils`、`common` 包。

新增相互独立的业务（例如 `users`、`billing`、`jobs`）时，创建同级包。只有当共享代码拥有清晰、可单独描述的职责时，才提取新包。

## 前端边界

- `api` 统一处理服务地址、响应错误与传输类型。
- `features/<domain>` 放置 Pinia store 和领域组件。
- `views` 只组合页面，不直接实现网络请求。
- `components/ui` 是 shadcn-vue CLI 管理的源码；升级时先用 `--dry-run` 和 `--diff` 查看上游变化。

## 配置原则

所有运行时配置来自环境变量。敏感值不打包进镜像，也不提交 `.env`。`development` 环境允许开发专用 JWT secret，`production` 环境的配置加载会拒绝空值或短 secret。

## 下一步占位

模板有意不预装项目尚未需要的复杂基础设施。常见扩展位置如下：

- 用户与登录：新增 `server/users`，在 `auth` 上组合 HTTP 中间件。
- 后台任务：新增 `server/jobs`，让每个 goroutine 都由 context 或关闭 channel 管理生命周期。
- 对象存储：新增 `server/storage`，以具体客户端开始，出现多个实现后再由消费方定义接口。
- 前端页面：新增 `web/src/features/<domain>` 与 `web/src/views/<Name>View.vue`，再注册路由。
- 生产部署：在 Release 镜像发布完成后接入目标平台，不把服务器凭证写入仓库。
