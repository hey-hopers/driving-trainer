package com.heyhopers.drivingtrainer.data

interface HealthRepository {
    suspend fun checkBackend(): BackendHealth
}

data class BackendHealth(
    val isHealthy: Boolean,
    val status: String,
)
