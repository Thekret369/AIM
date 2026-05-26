package com.lanline.app.core.ui

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

object LanLineColors {
    val Primary = Color(0xFF416BAA)
    val PrimaryStrong = Color(0xFF274D82)
    val PrimarySoft = Color(0xFFEAF2FF)
    val Accent = Color(0xFF6DB7B1)
    val AccentSoft = Color(0xFFE8F7F5)
    val Ai = Color(0xFF2F6F9F)
    val AiSoft = Color(0xFFE6F3FA)
    val Background = Color(0xFFF7F9FC)
    val Surface = Color(0xFFFFFFFF)
    val SurfaceAlt = Color(0xFFF0F5FA)
    val Text = Color(0xFF101828)
    val Muted = Color(0xFF667085)
    val Subtle = Color(0xFF98A2B3)
    val Line = Color(0xFFDCE5EF)
    val Warning = Color(0xFFF59E0B)
    val Danger = Color(0xFFD92D20)
    val Dark = Color(0xFF111827)
}

private val LanLineColorScheme = lightColorScheme(
    primary = LanLineColors.Primary,
    onPrimary = Color.White,
    secondary = LanLineColors.Accent,
    tertiary = LanLineColors.Ai,
    background = LanLineColors.Background,
    onBackground = LanLineColors.Text,
    surface = LanLineColors.Surface,
    onSurface = LanLineColors.Text,
    outline = LanLineColors.Line,
    error = LanLineColors.Danger,
)

private val LanLineTypography = Typography(
    titleLarge = TextStyle(fontSize = 22.sp, lineHeight = 30.sp, fontWeight = FontWeight.Bold),
    titleMedium = TextStyle(fontSize = 17.sp, lineHeight = 24.sp, fontWeight = FontWeight.Bold),
    bodyLarge = TextStyle(fontSize = 15.sp, lineHeight = 22.sp, fontWeight = FontWeight.Normal),
    bodyMedium = TextStyle(fontSize = 14.sp, lineHeight = 20.sp, fontWeight = FontWeight.Normal),
    bodySmall = TextStyle(fontSize = 12.sp, lineHeight = 18.sp, fontWeight = FontWeight.Normal),
    labelLarge = TextStyle(fontSize = 14.sp, lineHeight = 20.sp, fontWeight = FontWeight.Bold),
    labelMedium = TextStyle(fontSize = 12.sp, lineHeight = 18.sp, fontWeight = FontWeight.SemiBold),
)

private val LanLineShapes = Shapes(
    extraSmall = RoundedCornerShape(6.dp),
    small = RoundedCornerShape(8.dp),
    medium = RoundedCornerShape(8.dp),
    large = RoundedCornerShape(12.dp),
    extraLarge = RoundedCornerShape(20.dp),
)

@Composable
fun LanLineTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = LanLineColorScheme,
        typography = LanLineTypography,
        shapes = LanLineShapes,
        content = content,
    )
}
