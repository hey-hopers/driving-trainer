package com.heyhopers.drivingtrainer.ui.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.heyhopers.drivingtrainer.data.HealthRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class HomeViewModel @Inject constructor(
    private val healthRepository: HealthRepository,
) : ViewModel() {
    private val _uiState = MutableStateFlow(HomeUiState())
    val uiState: StateFlow<HomeUiState> = _uiState.asStateFlow()

    init {
        checkBackend()
    }

    fun checkBackend() {
        viewModelScope.launch {
            _uiState.update {
                it.copy(
                    isLoading = true,
                    message = "Checking local API...",
                )
            }

            runCatching { healthRepository.checkBackend() }
                .onSuccess { health ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            isHealthy = health.isHealthy,
                            message = if (health.isHealthy) {
                                "Connected to local API"
                            } else {
                                "Unexpected status: ${health.status}"
                            },
                        )
                    }
                }
                .onFailure { error ->
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            isHealthy = false,
                            message = error.message ?: "Backend is unreachable",
                        )
                    }
                }
        }
    }
}

data class HomeUiState(
    val isLoading: Boolean = false,
    val isHealthy: Boolean = false,
    val message: String = "Connection has not been checked yet",
)
