package com.heyhopers.drivingtrainer

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import com.heyhopers.drivingtrainer.ui.DrivingTrainerApp
import com.heyhopers.drivingtrainer.ui.theme.DrivingTrainerTheme
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            DrivingTrainerTheme {
                DrivingTrainerApp()
            }
        }
    }
}
