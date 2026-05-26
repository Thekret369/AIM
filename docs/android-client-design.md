# LanLine 安卓端测试包设计方案

## 目标

LanLine 安卓端第一阶段只做测试包，不追求正式上架能力。目标是把现有 Go 服务端能力接入到安卓竖屏触摸端，先验证多人即时通讯、AI 助手和文件上传在移动端的核心体验。

本阶段不考虑正式推送，不接入厂商推送或 FCM。

## 版本边界

### V0.1 UI 测试包

- 使用 Kotlin + Jetpack Compose 新建安卓工程。
- 先实现本地 UI 和页面跳转。
- 可配置测试服地址。
- 登录页可输入账号密码。
- 会话列表、单聊、AI 助手、联系人、我的页面可浏览。
- 聊天页支持本地模拟发送消息。
- 上传面板支持打开、关闭和入口选择反馈。
- 不要求真实后端联调。

### V0.2 接口联调包

- 登录接口接入 `/api/login`。
- 保存 JWT token 和服务器地址。
- 会话、联系人、群组等数据开始接入真实接口。
- WebSocket 接入 `/ws`。
- 单聊消息支持真实发送和接收。
- 文件上传接入 `/api/upload`。

### V0.3 稳定测试包

- 支持断线重连。
- 支持 token 过期后的重新登录。
- 支持上传失败重试提示。
- 支持基础日志采集。
- 支持 Test Android Apps 插件跑模拟器验收。

## 技术路线

| 模块 | 建议 |
| --- | --- |
| UI | Jetpack Compose |
| 语言 | Kotlin |
| 架构 | MVVM + Repository |
| HTTP | Retrofit + OkHttp |
| WebSocket | OkHttp WebSocket |
| 本地配置 | DataStore |
| 本地缓存 | Room，V0.1 暂不启用 |
| 图片加载 | Coil |
| JSON | Kotlinx Serialization 或 Moshi |
| 日志 | Timber 或 Android Log，后续再统一 |

## 目录结构建议

```text
android/
|-- app/
|   |-- src/main/java/com/lanline/app/
|   |   |-- MainActivity.kt
|   |   |-- LanLineApp.kt
|   |   |-- core/
|   |   |   |-- config/
|   |   |   |-- network/
|   |   |   |-- storage/
|   |   |   `-- ui/
|   |   |-- feature/
|   |   |   |-- setup/
|   |   |   |-- auth/
|   |   |   |-- chats/
|   |   |   |-- chat/
|   |   |   |-- ai/
|   |   |   |-- contacts/
|   |   |   `-- profile/
|   |   `-- model/
|   `-- build.gradle.kts
|-- build.gradle.kts
`-- settings.gradle.kts
```

## 页面设计

### 1. 测试服设置页

用途：测试包首次启动时配置后端地址。

核心控件：

- 服务器地址输入框。
- WebSocket 地址输入框。
- 环境分段按钮：局域网、测试服、公网 HTTPS。
- 连接检查卡片。
- 保存并进入登录按钮。

状态：

- 未配置。
- 已配置但未检查。
- HTTP 可达。
- WebSocket 待登录。
- 上传待测试。

### 2. 登录页

用途：使用现有 LanLine 账号登录。

核心控件：

- 账号输入框。
- 密码输入框。
- 登录按钮。
- 注册入口，V0.1 可只做 UI。
- 当前服务器提示。

接口：

- V0.2 接入 `POST /api/login`。

### 3. 会话页

用途：作为移动端主入口。

核心控件：

- 顶部标题和在线状态。
- 搜索框。
- 会话筛选：全部、未读、群聊、AI。
- 会话列表。
- 新建会话按钮。
- 底部导航。

会话类型：

- 单聊。
- 群聊。
- AI 助手。
- 系统公告。

### 4. 聊天页

用途：展示单聊或群聊消息。

核心控件：

- 顶部返回、标题、在线状态、更多按钮。
- 消息流。
- 图片或文件消息预览。
- 输入框。
- 发送按钮。
- 附件按钮。

消息状态：

- 发送中。
- 已发送。
- 已读。
- 发送失败。
- 已撤回。

动效：

- 进入聊天页：右侧滑入，约 220ms。
- 消息发送：气泡从输入区上方轻微缩放进入，约 160ms。
- 长按消息：菜单从触点附近缩放出现，约 140ms。

### 5. AI 助手页

用途：对接项目内置 AI 助手。

核心控件：

- 助手状态卡片。
- 快捷指令：总结聊天、生成测试用例、解释报错。
- AI 流式回复气泡。
- 引用上下文卡片。
- 输入框。

接口：

- V0.2 复用服务端 AI 消息链路。
- WebSocket 中处理 `ai_stream` 类型。

### 6. 联系人页

用途：查看好友、群组、好友申请。

核心控件：

- 搜索框。
- 分段按钮：好友、群组、申请。
- 常用联系人列表。
- 群组列表。

接口：

- `GET /api/friends`
- `GET /api/groups`
- `GET /api/friends/pending`

### 7. 我的页

用途：测试包状态和个人入口。

核心控件：

- 用户信息卡片。
- 服务状态卡片。
- AI 助手配置入口。
- 消息与存储入口。
- 安全入口。

状态展示：

- HTTP API 正常。
- WebSocket 已连接。
- 上传待测或正常。

### 8. 上传面板

用途：聊天页附件入口。

核心控件：

- 相册。
- 拍照。
- 文件。
- 语音，V0.1 可只占位。
- 取消按钮。

动效：

- 底部 Sheet 上滑进入，约 220ms。
- 背景遮罩渐显，约 180ms。
- 关闭时下滑并透明隐藏。

## 视觉规范

| 类型 | 取值 |
| --- | --- |
| 主色 | `#0E7C66` |
| AI 色 | `#6D5BD0` |
| 辅助蓝 | `#2563EB` |
| 背景 | `#F6F7F9` |
| 卡片 | `#FFFFFF` |
| 主文字 | `#101828` |
| 次文字 | `#667085` |
| 边框 | `#E4E7EC` |
| 圆角 | 8dp 为主 |
| 页面宽度 | 360dp 到 430dp 自适应 |

字体建议：

- Android 默认使用系统字体。
- 标题 20sp 到 22sp。
- 正文 14sp。
- 辅助文字 12sp。
- 按钮文字 14sp 到 15sp，使用 Medium 或 Bold。

## 状态模型

### AppConfig

```kotlin
data class AppConfig(
    val serverUrl: String,
    val websocketUrl: String,
    val environment: EnvironmentType
)
```

### AuthState

```kotlin
sealed interface AuthState {
    data object Anonymous : AuthState
    data class LoggedIn(val token: String, val user: LanLineUser) : AuthState
    data class Expired(val reason: String) : AuthState
}
```

### Conversation

```kotlin
data class Conversation(
    val id: Long,
    val type: ConversationType,
    val title: String,
    val avatarText: String,
    val lastMessage: String,
    val unreadCount: Int,
    val updatedAtText: String,
    val online: Boolean
)
```

### ChatMessage

```kotlin
data class ChatMessage(
    val id: Long,
    val type: MessageType,
    val content: String,
    val fromMe: Boolean,
    val status: MessageStatus,
    val createdAtText: String
)
```

## 后端接口映射

| 安卓功能 | 服务端接口 |
| --- | --- |
| 注册 | `POST /api/register` |
| 登录 | `POST /api/login` |
| 退出 | `POST /api/logout` |
| 个人信息 | `GET /api/profile` |
| 好友列表 | `GET /api/friends` |
| 群组列表 | `GET /api/groups` |
| 单聊历史 | `GET /api/history?peer_id=` |
| 群聊历史 | `GET /api/history/group/:id` |
| 消息搜索 | `GET /api/search/messages` |
| 文件上传 | `POST /api/upload` |
| WebSocket | `GET /ws?token=` |

## WebSocket 设计

当前服务端 WebSocket 消息类型：

- `chat`
- `typing`
- `read_receipt`
- `ai_stream`
- `status`
- `ack`
- `error`

安卓端建议封装：

```text
LanLineWebSocketClient
|-- connect(token)
|-- disconnect()
|-- sendChat(message)
|-- sendTyping(payload)
|-- sendReadReceipt(payload)
`-- observeEvents()
```

V0.2 阶段可以继续使用 `?token=` 接入。后续稳定后，再考虑服务端支持 `Authorization` Header 鉴权。

## 导航结构

```text
SetupRoute
LoginRoute
MainRoute
|-- ChatsRoute
|-- ContactsRoute
|-- AiRoute
`-- ProfileRoute
ChatRoute(conversationId, conversationType)
```

底部导航只出现在主页面：

- 聊天。
- 联系人。
- AI。
- 我的。

## Test Android Apps 验收清单

等安卓工程可打包后，用模拟器验证以下流程：

1. 安装测试包。
2. 启动 App。
3. 输入测试服地址。
4. 保存配置进入登录页。
5. 输入账号密码。
6. 登录后进入会话页。
7. 点击会话进入聊天页。
8. 输入并发送一条消息。
9. 打开上传面板。
10. 点击文件入口。
11. 返回会话页。
12. 切换到 AI 页面。
13. 切换到联系人页面。
14. 切换到我的页面。
15. 捕获截图和 logcat。

模拟器测试命令参考：

```bash
adb devices
./gradlew :app:installDebug --console=plain
adb -s <serial> shell cmd package resolve-activity --brief com.lanline.app
adb -s <serial> shell am start -n com.lanline.app/.MainActivity
adb -s <serial> exec-out screencap -p > /tmp/lanline-android.png
adb -s <serial> logcat -d > /tmp/lanline-logcat.txt
```

## 风险点

1. WebSocket 当前使用 query token，移动端测试可用，但日志中可能暴露 token。
2. Android 13 以上文件和图片权限需要单独处理。
3. HTTP 明文访问需要 debug 环境网络安全配置。
4. 局域网测试时，手机和服务器必须在同一网络，且服务器防火墙放通端口。
5. WebView 原型和 Compose 原生 UI 存在实现差异，后续应以 Compose 组件为准。

## 下一步

1. 确认是否采用 Kotlin + Jetpack Compose。
2. 确认安卓包名，例如 `com.lanline.app`。
3. 确认最低 Android 版本。
4. 确认 V0.1 是否只做本地 UI，还是直接接入登录接口。
5. 确认后新建 `android/` 工程。
