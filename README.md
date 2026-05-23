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

复制环境变量样例并填写 `AIM_JWT_SECRET`。

```powershell
Copy-Item .env.example .env
docker compose up -d --build
```

部署默认使用 SQLite：

- 数据库文件：`/app/data/aim.db`
- 上传文件：`/app/data/uploads`
- 日志目录：`/app/logs`

`docker-compose.yml` 使用命名卷持久化数据和日志。当前运行时代码只接入 SQLite，`AIM_DATABASE_DRIVER` 请保持为 `sqlite`。

## 常用环境变量

| 变量 | 说明 |
| --- | --- |
| `AIM_PORT` | 宿主机暴露端口，默认 `8080` |
| `AIM_JWT_SECRET` | JWT 签名密钥，必填 |
| `AIM_JWT_EXPIRE_HOURS` | 登录有效期小时数，默认 `72` |
| `AIM_AI_ENABLED` | 是否启用 AI 助手，默认 `true` |
| `AIM_AI_BASE_URL` | OpenAI 兼容接口地址 |
| `AIM_AI_API_KEY` | 默认 AI API Key |
| `AIM_AI_API_KEY_ENCRYPT_KEY` | AI Key 加密存储密钥，未设置时回退到 JWT 密钥 |
