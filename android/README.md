# LanLine Android

V0.4 LanLine 安卓后端联调测试包工程。

## 当前范围

- Kotlin + Jetpack Compose。
- 登录页已接真实后端 `POST /api/login`。
- 登录后接入真实后端 WebSocket `/ws?token=`。
- Android 13+ 会申请通知权限，在线收到新消息时弹本地通知。
- 已接入后端安卓版本检查接口 `GET /api/app/android/latest`。
- 默认测试服为 `http://47.109.109.164:8080`。
- 会话、聊天、联系人仍保留本地样例数据，并叠加实时消息入口。
- 暂不接正式厂商推送 SDK，但已预留 Push token 上报和推送设置同步接口。

## 页面

- 测试服设置。
- 登录。
- 会话。
- 聊天。
- AI 助手。
- 联系人。
- 我的。
- 上传面板。

## 预留接口

- `core/realtime/LanLineRealtimeClient.kt`：WebSocket 登录后连接、消息解析、断线重连。
- `core/notification/LocalMessageNotifier.kt`：本地通知渠道和消息提醒。
- `core/push/PushGateway.kt`：设备推送注册、Push token 上报、推送开关同步。
- `core/update/AppUpdateGateway.kt`：测试包版本检查、下载地址、SHA-256 校验值、强更策略。
- `core/auth/AuthRepository.kt`：公网测试服登录连调。
- `core/config/ServerConfig.kt`：把 `/login` 或 `/api/login` 地址规范化为服务根地址。

## 构建前置

当前仓库环境未检测到 JDK、Gradle、Android SDK。安装 Android Studio 后，打开 `android/` 目录，等待 Gradle 同步，再执行：

```bash
./gradlew :app:assembleDebug
```

或在 Windows PowerShell 中执行：

```powershell
.\gradlew.bat :app:assembleDebug
```

如果没有 Gradle Wrapper，请先在安装 Gradle 的环境中生成：

```bash
gradle wrapper --gradle-version 8.14.5
```
