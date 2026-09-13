package com.heyhopers.drivingtrainer.data.network

import retrofit2.http.GET

interface HealthApi {
    @GET("health")
    suspend fun getHealth(): HealthResponse
}
