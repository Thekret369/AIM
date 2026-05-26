package com.lanline.app.feature.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lanline.app.core.auth.AuthRepository
import com.lanline.app.core.auth.AuthSession
import com.lanline.app.core.config.ServerConfig
import com.lanline.app.core.ui.LanLineColors
import com.lanline.app.core.ui.LanLinePrimaryButton
import com.lanline.app.core.ui.LanLineTextField
import kotlinx.coroutines.launch

@Composable
fun LoginScreen(
    config: ServerConfig,
    onLogin: (AuthSession) -> Unit,
    onChangeServer: () -> Unit,
) {
    var username by rememberSaveable { mutableStateOf("zhangsan") }
    var password by rememberSaveable { mutableStateOf("password") }
    var errorMessage by rememberSaveable { mutableStateOf("") }
    var isLoggingIn by rememberSaveable { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val authRepository = remember(config) { AuthRepository(config) }

    fun submitLogin() {
        val cleanUsername = username.trim()
        if (cleanUsername.isBlank() || password.isBlank()) {
            errorMessage = "请填写账号和密码"
            return
        }
        if (isLoggingIn) return

        isLoggingIn = true
        errorMessage = ""
        scope.launch {
            authRepository.login(cleanUsername, password)
                .onSuccess { session ->
                    isLoggingIn = false
                    onLogin(session)
                }
                .onFailure { error ->
                    isLoggingIn = false
                    errorMessage = error.message ?: "登录失败，请检查账号或服务器"
                }
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(LanLineColors.Background)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 32.dp, vertical = 88.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = "LL",
                modifier = Modifier
                    .background(LanLineColors.Primary, RoundedCornerShape(8.dp))
                    .padding(horizontal = 17.dp, vertical = 15.dp),
                color = LanLineColors.Surface,
                fontSize = 18.sp,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.width(14.dp))
            Column {
                Text("LanLine", color = LanLineColors.Text, fontSize = 30.sp, fontWeight = FontWeight.Bold)
                Text("通讯 + AI 的移动测试入口", color = LanLineColors.Muted, fontSize = 13.sp)
            }
        }

        Spacer(Modifier.height(76.dp))
        LanLineTextField(label = "账号", value = username, onValueChange = { username = it })
        Spacer(Modifier.height(14.dp))
        // 连调包只在内存中保存输入内容，不做密码持久化。
        OutlinedTextField(
            value = password,
            onValueChange = { password = it },
            modifier = Modifier.fillMaxWidth(),
            label = { Text("密码") },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            shape = RoundedCornerShape(8.dp),
        )
        Spacer(Modifier.height(20.dp))
        LanLinePrimaryButton(
            text = if (isLoggingIn) "登录中" else "登录",
            enabled = !isLoggingIn,
            onClick = { submitLogin() },
        )
        if (errorMessage.isNotBlank()) {
            Text(
                text = errorMessage,
                color = LanLineColors.Danger,
                fontSize = 12.sp,
                modifier = Modifier.padding(top = 10.dp),
            )
        }
        Spacer(Modifier.height(12.dp))
        OutlinedButton(
            onClick = {},
            modifier = Modifier
                .fillMaxWidth()
                .height(48.dp),
            shape = RoundedCornerShape(8.dp),
        ) {
            Text("注册新账号", color = LanLineColors.Primary, fontWeight = FontWeight.Bold)
        }
        Spacer(Modifier.height(112.dp))
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(LanLineColors.SurfaceAlt, RoundedCornerShape(8.dp))
                .padding(horizontal = 14.dp, vertical = 13.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "当前服务器 ${config.apiBaseUrl}",
                modifier = Modifier.weight(1f),
                color = LanLineColors.Muted,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = "修改",
                color = LanLineColors.Primary,
                fontSize = 12.sp,
                fontWeight = FontWeight.Bold,
                modifier = Modifier
                    .clickable(onClick = onChangeServer)
                    .padding(start = 12.dp),
            )
        }
    }
}
