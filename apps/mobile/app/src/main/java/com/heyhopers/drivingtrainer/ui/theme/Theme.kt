package com.heyhopers.drivingtrainer.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable

private val LightColors = lightColorScheme(
    primary = TrainerGreen,
    secondary = TrainerBlue,
    error = TrainerRed,
    surface = TrainerSurface,
    surfaceContainer = TrainerSurfaceVariant,
)

@Composable
fun DrivingTrainerTheme(
    content: @Composable () -> Unit,
) {
    MaterialTheme(
        colorScheme = LightColors,
        content = content,
    )
}
