# AIM

AIM 是一个面向多人在线的即时通讯系统，内置可自部署的 AI 助手，将大模型能力集成到聊天场景中。

## 本地启动

启动前需要设置 JWT 密钥，否则配置校验会拒绝默认示例密钥。

```powershell
$env:AIM_JWT_SECRET="replace-with-a-long-random-secret"
go run ./cmd/server
```

默认访问地址：

```text
http://127.0.0.1:8080
```

## Docker Compose

Windows 推荐在 WSL 中进入项目目录后执行 Docker 命令，避免路径和换行符差异。

```bash
cd /mnt/d/GOtest/AIM
cp .env.example .env
```

编辑 `.env`，至少填写 `AIM_JWT_SECRET`，然后启动默认 SQLite 部署：

```bash
docker compose up -d --build
```

查看运行状态：

```bash
docker compose ps
docker compose logs -f aim
```

默认访问地址：

```text
http://127.0.0.1:8080
```

默认部署使用 SQLite 和命名卷：

- 数据库文件：`/app/data/aim.db`
- 上传文件：`/app/data/uploads`
- 日志目录：`/app/logs`
- 数据卷：`aim_data`
- 日志卷：`aim_logs`
- 网络：`aim_net`

镜像构建采用多阶段构建：第一阶段使用 Go 编译静态 Linux 二进制，运行阶段使用 Alpine、非 root 用户、只读根文件系统、健康检查和日志滚动配置。

## MySQL 部署

需要 MySQL 时，使用 MySQL 覆盖文件启动：

```bash
docker compose -f docker-compose.yml -f docker-compose.mysql.yml up -d --build
```

默认 MySQL DSN：

```env
AIM_DATABASE_DSN=aim:aim_local_password@tcp(mysql:3306)/aim?charset=utf8mb4&parseTime=True&loc=Local
```

如果修改 `.env` 中的 `AIM_MYSQL_PASSWORD` 或 `AIM_MYSQL_DATABASE`，需要同步修改 `AIM_DATABASE_DSN`。MySQL 服务带健康检查，`aim` 会等待 `mysql` 健康后启动。

MySQL 数据卷：

```text
aim_mysql
```

## 常用命令

```bash
# 构建镜像
docker compose build

# 启动 SQLite 部署
docker compose up -d

# 启动 MySQL 部署
docker compose -f docker-compose.yml -f docker-compose.mysql.yml up -d

# 查看服务健康状态
docker compose ps

# 查看应用日志
docker compose logs -f aim

# 停止服务，保留数据卷
docker compose down

# 停止服务并删除数据卷
docker compose down -v
```

## 常用环境变量

| 变量 | 说明 |
| --- | --- |
| `AIM_PORT` | 宿主机暴露端口，默认 `8080` |
| `AIM_JWT_SECRET` | JWT 签名密钥，必填 |
| `AIM_JWT_EXPIRE_HOURS` | 登录有效期小时数，默认 `72` |
| `AIM_DATABASE_DRIVER` | 数据库类型，默认 `sqlite`，可选 `mysql` |
| `AIM_DATABASE_DSN` | 数据库连接地址 |
| `AIM_LOG_LEVEL` | 日志级别，默认 `info` |
| `AIM_AI_ENABLED` | 是否启用 AI 助手，默认 `true` |
| `AIM_AI_BASE_URL` | OpenAI 兼容接口地址 |
| `AIM_AI_API_KEY` | 默认 AI API Key |
| `AIM_AI_API_KEY_ENCRYPT_KEY` | AI Key 加密存储密钥，未设置时回退到 JWT 密钥 |
