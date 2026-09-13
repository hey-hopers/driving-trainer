package com.heyhopers.drivingtrainer.ui

import androidx.compose.runtime.Composable
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import com.heyhopers.drivingtrainer.ui.home.HomeRoute

@Composable
fun DrivingTrainerApp() {
    val navController = rememberNavController()

    NavHost(
        navController = navController,
        startDestination = "home",
    ) {
        composable("home") {
            HomeRoute()
        }
    }
}
