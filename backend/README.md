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
Get-Content .\backend\migrations\000003_add_route_segment_road_use.up.sql | docker compose exec -T postgres psql -U $env:POSTGRES_USER -d $env:POSTGRES_DB
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
Quando o Valhalla local tiver DEM de elevacao disponivel, a resposta tambem inclui metricas de inclinacao nos segmentos e eventos `HILL` / `STEEP_HILL`.
A resposta tambem pode incluir eventos de curva detectados pela geometria da rota: `CURVE`, `SHARP_CURVE` e `CURVE_SEQUENCE`.
A resposta tambem pode incluir eventos viarios detectados por atributos do Valhalla e por transicoes entre segmentos: `INTERSECTION`, `COMPLEX_INTERSECTION`, `ROUNDABOUT`, `HIGHWAY_ENTRY` e `HIGHWAY_EXIT`.
A resposta tambem pode incluir eventos compostos detectados por associacao deterministica entre eventos proximos: `HILL_STOP` e `COMPLEX_INTERSECTION`.
`HILL_STOP` pode ser produzido por uma parada real proxima a uma ladeira ou pelo inicio/fim da rota ocorrer em uma ladeira, indicado por `metadata.stopContext` como `route_start` ou `route_end`.
A resposta inclui uma analise deterministica da dificuldade da rota com `overallDifficulty`, `averageDifficulty`, `peakDifficulty`, `complexityScore` e `categoryScores`. O campo legado `difficulty` continua presente com o mesmo valor de `overallDifficulty`.

Observacao: `STOP` e `TRAFFIC_LIGHT` existem como tipos de evento no dominio, mas dependem de dados que nao foram expostos pela integracao atual com `trace_attributes` durante a validacao local. Para esses eventos, pode ser necessario enriquecer a rota futuramente com Valhalla tiles, `/locate`, atributos customizados do Valhalla ou dados OSM/PostGIS.

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
    "engineVersion": "difficulty-v1",
    "difficulty": 3.42,
    "overallDifficulty": 3.42,
    "averageDifficulty": 2.84,
    "peakDifficulty": 6,
    "complexityScore": 3.75,
    "categoryScores": {
      "hills": 0,
      "curves": 0,
      "intersections": 3,
      "highSpeed": 6,
      "compound": 0,
      "eventDensity": 1.45,
      "eventDiversity": 2.4
    },
    "categories": {
      "hills": 0,
      "curves": 0,
      "intersections": 3,
      "highSpeed": 6
    }
  },
  "events": [
    {
      "type": "INTERSECTION",
      "position": {
        "latitude": -26.91,
        "longitude": -49.07
      },
      "routeDistanceMeters": 850,
      "difficultyScore": 3,
      "metadata": {
        "nodeType": "street_intersection",
        "intersectingEdges": 3
      }
    }
  ]
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
    "segments": [
      {
        "sequence": 0,
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
        "inclineAvgPercent": 3.2,
        "inclineMaxPercent": 4
      }
    ],
    "createdAt": "..."
  },
  "events": []
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

Expected response:

```json
{"status":"ok"}
```

The API also logs health checks in the terminal:

```text
health check received from 127.0.0.1:<port>
```
