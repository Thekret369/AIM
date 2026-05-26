package com.aim.app.core.ui

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

object AimColors {
    val Primary = Color(0xFF0E7C66)
    val PrimaryStrong = Color(0xFF075D4D)
    val PrimarySoft = Color(0xFFDFF6EF)
    val Accent = Color(0xFF2563EB)
    val AccentSoft = Color(0xFFEAF1FF)
    val Ai = Color(0xFF6D5BD0)
    val AiSoft = Color(0xFFF0EDFF)
    val Background = Color(0xFFF6F7F9)
    val Surface = Color(0xFFFFFFFF)
    val SurfaceAlt = Color(0xFFF1F5F9)
    val Text = Color(0xFF101828)
    val Muted = Color(0xFF667085)
    val Subtle = Color(0xFF98A2B3)
    val Line = Color(0xFFE4E7EC)
    val Warning = Color(0xFFF59E0B)
    val Danger = Color(0xFFD92D20)
    val Dark = Color(0xFF111827)
}

private val AimColorScheme = lightColorScheme(
    primary = AimColors.Primary,
    onPrimary = Color.White,
    secondary = AimColors.Accent,
    tertiary = AimColors.Ai,
    background = AimColors.Background,
    onBackground = AimColors.Text,
    surface = AimColors.Surface,
    onSurface = AimColors.Text,
    outline = AimColors.Line,
    error = AimColors.Danger,
)

private val AimTypography = Typography(
    titleLarge = TextStyle(fontSize = 22.sp, lineHeight = 30.sp, fontWeight = FontWeight.Bold),
    titleMedium = TextStyle(fontSize = 17.sp, lineHeight = 24.sp, fontWeight = FontWeight.Bold),
    bodyLarge = TextStyle(fontSize = 15.sp, lineHeight = 22.sp, fontWeight = FontWeight.Normal),
    bodyMedium = TextStyle(fontSize = 14.sp, lineHeight = 20.sp, fontWeight = FontWeight.Normal),
    bodySmall = TextStyle(fontSize = 12.sp, lineHeight = 18.sp, fontWeight = FontWeight.Normal),
    labelLarge = TextStyle(fontSize = 14.sp, lineHeight = 20.sp, fontWeight = FontWeight.Bold),
    labelMedium = TextStyle(fontSize = 12.sp, lineHeight = 18.sp, fontWeight = FontWeight.SemiBold),
)

private val AimShapes = Shapes(
    extraSmall = RoundedCornerShape(6.dp),
    small = RoundedCornerShape(8.dp),
    medium = RoundedCornerShape(8.dp),
    large = RoundedCornerShape(12.dp),
    extraLarge = RoundedCornerShape(20.dp),
)

@Composable
fun AimTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = AimColorScheme,
        typography = AimTypography,
        shapes = AimShapes,
        content = content,
    )
}
