# Valhalla local

Este documento prepara o Valhalla para o milestone M1 sem alterar o endpoint Go existente.

O Valhalla e tratado como infraestrutura local:

```text
Go API
  -> Routing abstraction
  -> Valhalla
  -> OpenStreetMap
```

Neste momento a API Go ainda retorna a analise mockada. A validacao abaixo testa somente a API HTTP do Valhalla.

## Dataset inicial

O `docker-compose.yml` usa a imagem oficial scripted:

```text
ghcr.io/valhalla/valhalla-scripted:3.8.3
```

O dataset inicial e regional, conforme a arquitetura recomenda:

```text
https://download.geofabrik.de/south-america/brazil/sul-latest.osm.pbf
```

Esse recorte cobre PR, SC e RS e serve para validar rotas locais de desenvolvimento sem baixar o Brasil inteiro.

Os arquivos gerados ficam em:

```text
infrastructure/valhalla/custom_files
```

Essa pasta e ignorada pelo Git porque recebe PBF, tiles, banco de areas administrativas, time zones, `valhalla.json` e o tarball de tiles.

## Subir Valhalla

Na raiz do projeto:

```powershell
docker compose up -d valhalla
```

Acompanhar o primeiro build:

```powershell
docker compose logs -f valhalla
```

Na primeira execucao o container baixa o PBF regional e constroi os tiles. Isso pode levar varios minutos. Reinicios posteriores tendem a reaproveitar os arquivos em `infrastructure/valhalla/custom_files`.

Para subir banco e Valhalla juntos:

```powershell
docker compose up -d
```

## Validar status

```powershell
Invoke-RestMethod http://localhost:8002/status
```

Resposta esperada: HTTP 200 com campos como `version` e `tileset_last_modified`.

## Validar uma rota diretamente no Valhalla

Exemplo com coordenadas em Blumenau/SC, dentro do recorte `sul-latest.osm.pbf`:

```powershell
$body = @{
  locations = @(
    @{
      lat = -26.9194
      lon = -49.0661
      type = "break"
    },
    @{
      lat = -26.9050
      lon = -49.0750
      type = "break"
    }
  )
  costing = "auto"
  directions_options = @{
    units = "kilometers"
  }
} | ConvertTo-Json -Depth 5

$response = Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8002/route" `
  -ContentType "application/json" `
  -Body $body

$response.trip.summary
$response.trip.legs[0].shape
```

Campos importantes para M1:

```text
trip.summary.length
trip.summary.time
trip.legs[0].shape
```

Observacoes:

* `trip.summary.length` vem em quilometros quando `directions_options.units` e `kilometers`.
* `trip.summary.time` vem em segundos.
* `trip.legs[0].shape` e uma polyline com precisao 6, conhecida como polyline6.

## Elevacao para M4

O `docker-compose.yml` deixa `build_elevation` habilitado para que o Valhalla construa dados de elevacao locais junto com os tiles:

```text
build_elevation: "True"
```

Com elevacao disponivel, a API Go chama tambem o endpoint `/height` do Valhalla usando a polyline6 da rota. O perfil retornado e usado para calcular:

* elevacao inicial/final dos segmentos;
* inclinacao media;
* inclinacao maxima;
* eventos `HILL`;
* eventos `STEEP_HILL`.

Se a pasta `infrastructure/valhalla/custom_files` ja tiver tiles criados sem elevacao, remova os arquivos gerados ou force um rebuild antes de validar a M4. O primeiro build com DEM pode levar mais tempo porque o container precisa baixar e preparar arquivos adicionais de elevacao.

Validacao direta do endpoint de elevacao:

```powershell
$shape = "<polyline6-retornada-pelo-endpoint-route>"

$body = @{
  encoded_polyline = $shape
  shape_format = "polyline6"
  range = $true
  resample_distance = 25
  height_precision = 1
} | ConvertTo-Json -Depth 3

Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8002/height" `
  -ContentType "application/json" `
  -Body $body
```

## Validar com curl

```bash
curl -X POST http://localhost:8002/route \
  -H "Content-Type: application/json" \
  -d '{
    "locations": [
      {
        "lat": -26.9194,
        "lon": -49.0661,
        "type": "break"
      },
      {
        "lat": -26.9050,
        "lon": -49.0750,
        "type": "break"
      }
    ],
    "costing": "auto",
    "directions_options": {
      "units": "kilometers"
    }
  }'
```

Com `jq`:

```bash
curl -s -X POST http://localhost:8002/route \
  -H "Content-Type: application/json" \
  -d '{
    "locations": [
      { "lat": -26.9194, "lon": -49.0661, "type": "break" },
      { "lat": -26.9050, "lon": -49.0750, "type": "break" }
    ],
    "costing": "auto",
    "directions_options": { "units": "kilometers" }
  }' | jq '.trip.summary, .trip.legs[0].shape'
```

## Troubleshooting

Se `/route` retornar erro 400:

* confirme que o build terminou nos logs;
* confirme que as coordenadas estao dentro da regiao Sul;
* use pontos proximos a ruas mapeadas no OpenStreetMap;
* consulte `/status` antes de testar a rota.

Se o container for encerrado durante o build:

* reduza `server_threads` no `docker-compose.yml`;
* aumente memoria/CPU disponivel para o Docker Desktop;
* remova arquivos incompletos de `infrastructure/valhalla/custom_files` e suba novamente, se o cache local tiver ficado inconsistente.

## Referencias

* Valhalla Docker scripted image: https://github.com/valhalla/valhalla/blob/master/docker/README.md
* Valhalla `/route` API: https://valhalla.github.io/valhalla/api/turn-by-turn/api-reference/
* Valhalla `/height` API: https://valhalla.github.io/valhalla/api/elevation/
* Valhalla `/status` API: https://valhalla.github.io/valhalla/api/status/
* Geofabrik Brazil/Sul OSM extract: https://download.geofabrik.de/south-america/brazil/sul.html
