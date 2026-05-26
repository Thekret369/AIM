package com.aim.app.feature.auth

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
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.aim.app.core.ui.AimColors
import com.aim.app.core.ui.AimPrimaryButton
import com.aim.app.core.ui.AimTextField

@Composable
fun LoginScreen(
    onLogin: () -> Unit,
    onChangeServer: () -> Unit,
) {
    var username by rememberSaveable { mutableStateOf("zhangsan") }
    var password by rememberSaveable { mutableStateOf("password") }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(AimColors.Background)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 32.dp, vertical = 88.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                text = "AI",
                modifier = Modifier
                    .background(AimColors.Primary, RoundedCornerShape(8.dp))
                    .padding(horizontal = 17.dp, vertical = 15.dp),
                color = AimColors.Surface,
                fontSize = 18.sp,
                fontWeight = FontWeight.Bold,
            )
            Spacer(Modifier.width(14.dp))
            Column {
                Text("AIM", color = AimColors.Text, fontSize = 30.sp, fontWeight = FontWeight.Bold)
                Text("通讯 + AI 的移动测试入口", color = AimColors.Muted, fontSize = 13.sp)
            }
        }

        Spacer(Modifier.height(76.dp))
        AimTextField(label = "账号", value = username, onValueChange = { username = it })
        Spacer(Modifier.height(14.dp))
        // V0.1 只做 UI，密码仍保存在本地状态中，不写入持久化存储。
        androidx.compose.material3.OutlinedTextField(
            value = password,
            onValueChange = { password = it },
            modifier = Modifier.fillMaxWidth(),
            label = { Text("密码") },
            singleLine = true,
            visualTransformation = PasswordVisualTransformation(),
            shape = RoundedCornerShape(8.dp),
        )
        Spacer(Modifier.height(20.dp))
        AimPrimaryButton(text = "登录", onClick = onLogin)
        Spacer(Modifier.height(12.dp))
        OutlinedButton(
            onClick = {},
            modifier = Modifier
                .fillMaxWidth()
                .height(48.dp),
            shape = RoundedCornerShape(8.dp),
        ) {
            Text("注册新账号", color = AimColors.Primary, fontWeight = FontWeight.Bold)
        }
        Spacer(Modifier.height(112.dp))
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(AimColors.SurfaceAlt, RoundedCornerShape(8.dp))
                .padding(horizontal = 14.dp, vertical = 13.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = "当前服务器 192.168.1.8:8080",
                modifier = Modifier.weight(1f),
                color = AimColors.Muted,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = "修改",
                color = AimColors.Primary,
                fontSize = 12.sp,
                fontWeight = FontWeight.Bold,
                modifier = Modifier
                    .clickable(onClick = onChangeServer)
                    .padding(start = 12.dp),
            )
        }
    }
}
