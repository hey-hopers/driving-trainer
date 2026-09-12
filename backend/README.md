# Driving Trainer Backend

API Go para analise de rotas do projeto Driving Trainer.

## Instalar dependencias

```powershell
cd backend
go mod tidy
```

## Executar a API

Antes de chamar a analise de rota real, suba o Valhalla local conforme `docs/valhalla-local.md`.
Tambem e necessario aplicar as migrations no PostgreSQL/PostGIS e configurar `DATABASE_URL`.

Exemplo de variavel de conexao:

```powershell
$env:DATABASE_URL = "postgres://<usuario>:<senha>@localhost:5432/<database>?sslmode=disable"
```

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

Para apontar para outro Valhalla:

```powershell
$env:VALHALLA_URL = "http://localhost:8002"
go run ./cmd/api
```

## Executar migrations

As migrations ficam em `backend/migrations`.

Com `golang-migrate`:

```powershell
cd backend
migrate -path migrations -database $env:DATABASE_URL up
```

Alternativa com `psql` no container Docker:

```powershell
cd ..
Get-Content .\backend\migrations\000001_init.up.sql | docker compose exec -T postgres psql -U $env:POSTGRES_USER -d $env:POSTGRES_DB
Get-Content .\backend\migrations\000002_add_route_original_polyline.up.sql | docker compose exec -T postgres psql -U $env:POSTGRES_USER -d $env:POSTGRES_DB
cd backend
```

## Executar os testes

```powershell
cd backend
go test ./...
```

## Regenerar queries sqlc

```powershell
cd backend
sqlc generate
```

## Testar a analise de rota

O endpoint chama o Valhalla e retorna distancia, duracao e polyline real da rota.

Observacao: `route.polyline` contem uma polyline6 retornada pelo Valhalla.

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

Resposta esperada:

```json
{
  "route": {
    "id": "...",
    "distanceMeters": 2476,
    "durationSeconds": 376,
    "polyline": "..."
  },
  "analysis": {
    "difficulty": 0,
    "categories": {
      "hills": 0,
      "curves": 0,
      "intersections": 0,
      "highSpeed": 0
    }
  },
  "events": []
}
```

O `id` retornado pode ser usado para recuperar a rota persistida.

## Buscar rota persistida

### PowerShell

```powershell
Invoke-RestMethod "http://localhost:8080/api/v1/routes/<route-id>"
```

### curl

```bash
curl http://localhost:8080/api/v1/routes/<route-id>
```

Resposta esperada:

```json
{
  "route": {
    "id": "...",
    "source": "valhalla",
    "geometry": [
      {
        "latitude": -26.9194,
        "longitude": -49.0661
      },
      {
        "latitude": -26.905,
        "longitude": -49.075
      }
    ],
    "distanceMeters": 2476,
    "durationSeconds": 376,
    "polyline": "...",
    "createdAt": "..."
  }
}
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
