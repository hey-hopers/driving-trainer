# Driving Trainer Backend

API Go para analise de rotas do projeto Driving Trainer.

## Instalar dependencias

```powershell
cd backend
go mod tidy
```

## Executar a API

```powershell
cd backend
go run ./cmd/api
```

Por padrao, a API sobe em:

```text
http://localhost:8080
```

Para alterar a porta:

```powershell
$env:PORT = "8081"
go run ./cmd/api
```

## Executar os testes

```powershell
cd backend
go test ./...
```

## Testar a analise de rota

### PowerShell

```powershell
$body = @{
  origin = @{
    latitude = -26.9194
    longitude = -49.0661
  }
  destination = @{
    latitude = -26.9050
    longitude = -49.0750
  }
} | ConvertTo-Json -Depth 3

Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8080/api/v1/routes/analyze" `
  -ContentType "application/json" `
  -Body $body
```

### curl

```bash
curl -X POST http://localhost:8080/api/v1/routes/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "origin": {
      "latitude": -26.9194,
      "longitude": -49.0661
    },
    "destination": {
      "latitude": -26.9050,
      "longitude": -49.0750
    }
  }'
```

## Health check

```powershell
Invoke-RestMethod http://localhost:8080/health
```
