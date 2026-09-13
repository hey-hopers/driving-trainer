package com.heyhopers.drivingtrainer.data

import com.heyhopers.drivingtrainer.data.network.HealthApi
import javax.inject.Inject

class RemoteHealthRepository @Inject constructor(
    private val healthApi: HealthApi,
) : HealthRepository {
    override suspend fun checkBackend(): BackendHealth {
        val response = healthApi.getHealth()
        return BackendHealth(
            isHealthy = response.status == "ok",
            status = response.status,
        )
    }
}
