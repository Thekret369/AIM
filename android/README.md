# AIM Android

V0.1 安卓 UI 测试包工程。

## 当前范围

- Kotlin + Jetpack Compose。
- 只做本地 UI 和页面跳转。
- 暂不接真实后端。
- 暂不接正式推送，但已预留 Push token 上报和推送设置同步接口。
- 已预留测试包更新检查接口，后续可接 APK 下载地址、校验值和强更策略。

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

- `core/push/PushGateway.kt`：设备推送注册、Push token 上报、推送开关同步。
- `core/update/AppUpdateGateway.kt`：测试包版本检查、下载地址、SHA-256 校验值、强更策略。

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
