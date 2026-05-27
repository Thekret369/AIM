# LanLine Android

V0.5 LanLine 安卓后端联调测试包工程。

## 当前范围

- Kotlin + Jetpack Compose 竖屏触摸端 UI。
- 登录页不再预填任何测试账号；登录走真实后端 `POST /api/login`。
- 默认测试服务为 `http://47.109.109.164:8080`，服务地址可在设置页修改。
- 登录后连接真实 WebSocket `/ws?token=`，支持实时消息、本地通知、断线重连、发送文本消息、输入状态与已读回执协议入口。
- 会话、联系人、AI 助手、群组、聊天历史优先读取后端接口，不再加载本地样例数据。
- 已接入安卓版本检查接口 `GET /api/app/android/latest`，当前测试包版本为 `0.5.0-debug`。
- 暂不接正式厂商 Push SDK，但保留 Push token 上报、推送设置同步和本地通知能力。

## 已封装网页端接口

- 认证：注册、登录、登出、个人资料、修改资料、修改密码。
- 好友：好友列表、在线好友、待处理申请、添加好友、处理申请、删除好友、修改备注。
- 分组：联系人分组增删改查。
- 群组：创建、列表、详情、编辑、加入、加人、退群、踢人、转让群主、管理员、禁言、免打扰、成员、已读、公告。
- 消息：单聊历史、群聊历史、广播历史、搜索、撤回、删除。
- AI：机器人列表、创建、更新、删除、Token 用量、知识库、知识文档增删改。
- 设置：用户设置读取与更新。
- 上传：`POST /api/upload` 文件上传入口。
- 版本：安卓最新版本检查。

## 页面

- 测试服设置。
- 登录。
- 会话。
- 聊天。
- AI 助手。
- 联系人。
- 我的。
- 上传面板。

## 关键文件

- `core/network/LanLineApiClient.kt`：带 Token 的 HTTP JSON 与上传客户端。
- `core/network/LanLineWebApi.kt`：网页端功能接口的安卓封装。
- `core/backend/LanLineBackendRepository.kt`：把后端好友、群组、AI 和历史消息转换为 Compose UI 数据。
- `core/realtime/LanLineRealtimeClient.kt`：WebSocket 登录后连接、消息解析、发送、断线重连。
- `core/notification/LocalMessageNotifier.kt`：本地通知渠道和消息提醒。
- `core/push/PushGateway.kt`：设备推送注册、Push token 上报、推送开关同步。
- `core/update/AppUpdateGateway.kt`：测试包版本检查、下载地址、SHA-256 校验值、强更策略。
- `core/auth/AuthRepository.kt`：公网测试服登录联调。
- `core/config/ServerConfig.kt`：把 `/login` 或 `/api/login` 地址规范化为服务根地址。

## 构建

当前仓库已在 `.codex_deploy/` 下准备 Android Studio、Android SDK 与 Gradle，示例 PowerShell 命令：

```powershell
$env:JAVA_HOME = (Resolve-Path .\.codex_deploy\android-studio\jbr).Path
$env:ANDROID_HOME = (Resolve-Path .\.codex_deploy\android-sdk).Path
$env:ANDROID_SDK_ROOT = $env:ANDROID_HOME
$env:Path = "$env:JAVA_HOME\bin;$env:ANDROID_HOME\platform-tools;$env:ANDROID_HOME\cmdline-tools\latest\bin;$PWD\.codex_deploy\gradle-8.14.5\bin;$env:Path"
.\.codex_deploy\gradle-8.14.5\bin\gradle.bat -p android :app:assembleDebug --console=plain
```
