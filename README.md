# XorAPI — AI API 中转站

基于 Go (Gin) + Vue 3 的高性能 AI API 中转站，架构参考 Sub2API 并针对国内虚拟主机环境深度定制。
**一个 API Key，同时调用 OpenAI / Anthropic 等所有大模型，协议自动双向转换。**

## 核心特性

- **协议转换**：`/v1/chat/completions`（OpenAI 风格）与 `/anthropic/v1/messages`（Anthropic 风格）双入口，内部自动双向转换，任意客户端 × 任意上游渠道自由组合
- **智能路由**：模型分组管理，组内多渠道按优先级重试，同名模型跨分组自动 fallback
- **点数计费**：每模型独立配置按 Token / 按次计费；扣费优先消耗即将过期的免费点数（签到所得），再扣充值点数，全程流水可查
- **Web 安装向导**：首次访问自动引导，环境检测 → 填写配置 → 一键安装，无需手动编辑配置文件
- **虚拟主机友好**：单二进制部署（前端已嵌入），Nginx 反代即用；MySQL 5.6.4+/5.7+/8.0+（兼容 MariaDB 10.x），Redis 可选（缺失时自动回退内存模式）
- **安全**：BCrypt 密码、JWT 认证、API Key 哈希存储、上游渠道 Key AES-256-GCM 加密、IP 白名单、每分钟限流、每日调用上限
- **前端模板系统**：后台一键切换站点外观，支持上传 ZIP 自定义模板（含开发规范与示例）
- **模型广场**：公开展示已接入模型，支持搜索、标签筛选、示例代码一键复制

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.23+ / Gin / GORM |
| 数据库 | MySQL 5.6.4+ / 5.7+ / 8.0+（utf8mb4，兼容 MariaDB 10.x） |
| 缓存 | Redis 7+（可选，用于限流与计数） |
| 前端 | Vue 3 / Vite 5 / TailwindCSS 3 / Pinia |
| 部署 | 单二进制 + Nginx 反向代理 |

> 与 Sub2API 原版差异：ORM 由 Ent 调整为 GORM（无需代码生成即可兼容 MySQL 迁移），新增点数计费、签到、模型分组路由、安装向导与模板系统。

## 目录结构

```
├── backend/                 # Go 后端
│   ├── cmd/server/main.go   # 入口
│   ├── internal/
│   │   ├── config/          # 配置加载与生成
│   │   ├── store/           # 全局 DB/Redis、限流计数
│   │   ├── model/           # GORM 数据模型
│   │   ├── middleware/      # JWT / API Key / 管理员鉴权
│   │   ├── handler/         # 业务与管理接口
│   │   ├── relay/           # 协议转换与中继核心
│   │   ├── service/         # 计费/路由/加密/签到/邮件
│   │   ├── router/          # 路由与静态资源
│   │   └── web/dist/        # 内嵌前端产物（构建时生成）
│   └── go.mod
├── frontend/                # Vue 3 前端（构建产物嵌入后端）
├── deploy/                  # 部署物料
│   ├── nginx.conf.example   # Nginx 配置（含 SSE/WebSocket 优化）
│   ├── config.example.yaml  # 配置说明
│   ├── xorapi.service       # systemd 服务
│   └── template-example/    # 自定义模板示例
└── build.sh                 # 一键构建脚本
```

## 快速开始（生产部署）

1. **下载二进制**：`release/xorapi-linux-amd64`（或自行构建，见下文）
2. **上传到服务器**，赋予执行权限：
   ```bash
   chmod +x xorapi-linux-amd64
   ./xorapi-linux-amd64 -port 8080
   ```
3. **配置 Nginx**：参考 `deploy/nginx.conf.example`（已含 `underscores_in_headers on`、SSE 关闭缓冲等关键配置）
4. **浏览器访问** `http://your-domain.com/install` 完成安装向导：
   - 环境检测：MySQL 连接（可自动建库）、Redis（可选）、目录权限
   - 填写管理员邮箱密码
   - 一键安装，自动生成 `config.yaml` 与数据表
5. 登录管理后台 → 创建分组与渠道 → 添加模型 → 用户即可调用

## 开发环境

```bash
# 首次克隆：一键完成前端构建、产物嵌入与后端编译
./build.sh

# 前端
cd frontend && pnpm install && pnpm dev     # http://localhost:5173（已代理 /api 到 8080）

# 后端
cd backend && go run ./cmd/server           # http://localhost:8080

# 构建（前端产物将嵌入二进制）
./build.sh
# 或手动：
cd frontend && pnpm build
cd backend && go build -o xorapi.exe ./cmd/server
# Linux 交叉编译
GOOS=linux GOARCH=amd64 go build -o xorapi-linux-amd64 ./cmd/server
```

> 本地开发后端无 MySQL 时可直接启动，访问 `/install` 走安装向导流程。

## API 一览

### 中继接口（API Key 认证，`Authorization: Bearer sk-xxx` 或 `x-api-key: sk-xxx`）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/v1/chat/completions` | OpenAI 兼容主入口（支持流式 SSE） |
| POST | `/anthropic/v1/messages` | Anthropic 兼容入口（支持流式 SSE） |
| GET | `/v1/models` | 可用模型列表 |

### 公开接口
`POST /api/auth/register` · `POST /api/auth/login` · `POST /api/auth/email-code` · `GET /api/square/models` · `GET /api/status`

### 用户接口（JWT）
`GET/PUT /api/user/profile` · `POST /api/user/password` · `POST /api/user/signin` · `GET /api/user/signin/status` · `GET /api/user/transactions` · `GET /api/user/logs` · `CRUD /api/keys`

### 管理接口（JWT + 管理员）
`GET/PUT /api/admin/configs` · `CRUD /api/admin/groups`（含渠道，全量替换） · `POST /api/admin/channels/:id/test` · `CRUD /api/admin/models` · `GET/PUT /api/admin/users` · `POST /api/admin/users/:id/points` · `CRUD /api/admin/invites` · `GET /api/admin/dashboard` · `GET /api/admin/templates` · `POST /api/admin/templates/upload` · `PUT /api/admin/templates/activate`

## 计费说明

- 每模型独立配置：`按 Token`（输入/输出分别计价，单位：点/百万 Token）或 `按次`（固定点数/次）
- **扣费顺序**：优先扣除**即将过期的免费点数**（签到奖励，默认 30 天有效），其次扣除充值点数；逐批扣减，事务内行锁保证并发安全
- 所有变动记录于交易流水；每次调用记录于调用日志（Token 数、耗时、渠道）

## 自定义模板开发

ZIP 根目录需包含 `manifest.json` 与 `index.html`，规范见后台"模板管理"页内说明，完整示例见 `deploy/template-example/`。
模板仅负责展示，数据通过同源接口（`/api/square/models`、`/api/status`）获取，所有 API 调用仍由后端处理，天然安全隔离。

## 安全说明

- 上游渠道 API Key 使用 AES-256-GCM 加密存储，密钥安装时自动生成于 `config.yaml`
- 用户 API Key 仅存 SHA-256 哈希，创建时明文只显示一次
- JWT 有效期 7 天；管理员接口双重校验
- 建议生产环境启用 HTTPS 并将 `config.yaml` 权限设为 600

## 常见问题

- **忘记密码/重装**：删除 `config.yaml` 与 `data/install.lock` 后重启，重新走安装向导（原数据保留在数据库中，管理员邮箱重复时需换邮箱或手动清理 users 表）
- **重置管理员**：`mysql> UPDATE users SET role='admin' WHERE email='你的邮箱';`
- **流式响应被截断**：确认 Nginx 配置了 `proxy_buffering off`
- **Redis 未部署**：可正常运行，限流与每日计数回退为进程内存模式（重启后计数清零）
