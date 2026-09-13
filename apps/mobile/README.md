# Driving Trainer Android

Primeira fundacao Android do projeto.

## Stack

* Kotlin
* Jetpack Compose
* Hilt
* Coroutines / Flow
* Retrofit / OkHttp

## Backend local

O app usa `http://10.0.2.2:8080/` por padrao para acessar o backend local a partir do Android Emulator.

Para testar em um dispositivo fisico, altere `API_BASE_URL` em `app/build.gradle.kts` para o IP da maquina na rede local, por exemplo:

```kotlin
buildConfigField("String", "API_BASE_URL", "\"http://192.168.0.10:8080/\"")
```

## Validacao

Com o backend rodando:

```powershell
cd apps/mobile
gradle :app:assembleDebug
```

Se o Gradle nao estiver instalado, abra `apps/mobile` no Android Studio e rode o sync/build por la.

Depois abra o app no Android Emulator e toque em `Check connection`. A tela deve mostrar `Connected to local API` quando `GET /health` responder `{"status":"ok"}`.

No terminal do backend, a validacao tambem deve aparecer como:

```text
health check received from 127.0.0.1:<port>
```
