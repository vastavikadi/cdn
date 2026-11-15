# Go CDN --- Origin and Edge Cache

A learning-focused CDN built with Go and an existing Next.js frontend.
The system has one Origin server and two independently running Edge
servers. Origin owns original assets and metadata; each Edge serves
assets from its own local disk cache.

## Architecture

``` mermaid
flowchart TB
    UI["Next.js frontend"]
    O["Go Origin :8080"]
    M[("data/assets.json")]
    F[("Original asset files")]
    I["India Edge :8081"]
    E["Europe Edge :8082"]
    IC[("India cache")]
    EC[("Europe cache")]

    UI -->|"Upload / list / delete"| O
    O --> M
    O --> F
    UI -->|"Selected delivery region"| I
    UI -->|"Selected delivery region"| E
    I -->|"MISS: fetch content"| O
    E -->|"MISS: fetch content"| O
    I --> IC
    E --> EC
    O -.->|"Deletion invalidation"| I
    O -.->|"Deletion invalidation"| E
```

## Request flows

### Upload

``` mermaid
sequenceDiagram
    participant UI as Next.js
    participant O as Origin
    participant FS as Asset storage
    participant DB as JSON metadata
    UI->>O: POST /api/assets/upload
    O->>FS: Write original file
    O->>DB: Persist asset metadata
    O-->>UI: 201 Created + asset details
```

### Delivery on cache miss

``` mermaid
sequenceDiagram
    participant UI as Next.js
    participant E as Selected Edge
    participant O as Origin
    UI->>E: GET /assets/{id}
    E->>E: Check local cache
    E->>O: GET /api/assets/{id}/content
    O-->>E: Bytes and content headers
    E->>E: Store body and metadata
    E-->>UI: Asset response (X-Cache: MISS)
```

### Delivery on cache hit

``` mermaid
sequenceDiagram
    participant UI as Next.js
    participant E as Selected Edge
    UI->>E: GET /assets/{id}
    E->>E: Find local cache entry
    E-->>UI: Local asset (X-Cache: HIT)
```

### Deletion invalidation

``` mermaid
sequenceDiagram
    participant UI as Next.js
    participant O as Origin :8080
    participant I as India Edge :8081
    participant E as Europe Edge :8082
    UI->>O: DELETE /api/assets/{id}
    O->>O: Delete original and metadata
    O->>I: POST /internal/cache/{id}/invalidate
    O->>E: POST /internal/cache/{id}/invalidate
    I->>I: Remove .body and .json
    E->>E: Remove .body and .json
    O-->>UI: Deletion result
```

Origin attempts to notify both Edges after successful deletion. The
internal Edge endpoint requires the `X-Edge-Token` header. If an Edge is
unavailable, the current Origin implementation logs the failure; durable
retry/reconciliation is future work.

## Representative project structure

``` text
cdn/
├── main.go
├── edge_invalidator.go
├── upload_handler.go
├── delete_asset.go
├── serve_origin_asset.go
├── metadata_store.go
├── data/
│   └── assets.json
├── assets/
├── internals/
│   └── models/
│       └── asset.go
├── edge/
│   ├── main.go
│   ├── config.go
│   ├── handler.go
│   ├── cache.go
│   ├── config/
│   │   ├── india.json
│   │   └── europe.json
│   └── cache/
│       ├── india/
│       └── europe/
└── [Next.js frontend]
```

The module import path used in the project code is
`github.com/vastavikadi/cdn`. This is a representative tree; the
repository may contain additional files.

## Requirements

- Go installed and available on `PATH`.
- Existing Next.js frontend and its package manager/runtime.
- The commands below assume Windows PowerShell and project root
  `C:\projects\cdn`.

Check Go:

``` powershell
go version
```

## Configuration

Example `edge/config/india.json`:

``` json
{
  "id": "edge-india",
  "region": "India",
  "port": "8081",
  "origin_url": "http://localhost:8080",
  "cache_dir": "./edge/cache/india",
  "invalidate_token": "REPLACE_WITH_LOCAL_SECRET",
  "cache_ttl": "24h",
  "max_cache_bytes": 524288000
}
```

Example `edge/config/europe.json` uses `edge-europe`, region `Europe`,
port `8082`, and `./edge/cache/europe`. The Origin URL, invalidation
token, TTL and capacity should match the intended local setup.

- `id`: Edge identifier.
- `region`: region label returned in response headers.
- `port`: HTTP listening port.
- `origin_url`: Origin base URL.
- `cache_dir`: Edge-specific cache directory.
- `invalidate_token`: must match the Origin invalidator token and be
  sent as `X-Edge-Token`.
- `cache_ttl`: Go duration, e.g. `30s`, `1h`, `24h`.
- `max_cache_bytes`: cache capacity in bytes; `524288000` is 500 MiB.

Use a private unpredictable secret outside local development. Do not
commit real secrets to a public repository.

## Run locally on Windows

Open three PowerShell terminals from `C:\projects\cdn`.

Terminal 1 --- Origin:

``` powershell
go run .
```

Terminal 2 --- India Edge:

``` powershell
go run .\edge --config .\edge\config\india.json
```

Terminal 3 --- Europe Edge:

``` powershell
go run .\edge --config .\edge\config\europe.json
```

The local ports are Origin `8080`, India `8081`, Europe `8082`.

Check the basic Edge health endpoints:

``` powershell
curl.exe -i http://localhost:8081/health
curl.exe -i http://localhost:8082/health
```

The health endpoint is currently a basic process-level response. It is
not yet an active monitor that checks Origin reachability or controls
failover.

## HTTP API

### Origin

  ----------------------------------------------------------------------------
  Method                  Path                         Purpose
  ----------------------- ---------------------------- -----------------------
  POST                    `/api/assets/upload`         Upload an asset using
                                                       multipart field `file`

  GET                     `/api/assets`                List assets

  GET                     `/api/assets/{id}`           Get asset metadata

  GET                     `/api/assets/{id}/content`   Serve original bytes

  DELETE                  `/api/assets/{id}`           Delete original and
                                                       metadata, then notify
                                                       Edges
  ----------------------------------------------------------------------------

### Edge

  -----------------------------------------------------------------------------------
  Method                  Path                                Purpose
  ----------------------- ----------------------------------- -----------------------
  GET                     `/assets/{id}`                      Deliver asset via cache

  GET                     `/health`                           Basic Edge health
                                                              response

  POST                    `/internal/cache/{id}/invalidate`   Remove cached asset;
                                                              requires `X-Edge-Token`
  -----------------------------------------------------------------------------------

The invalidation endpoint is for server-to-server use, not direct
browser use.

### Useful response headers

- `X-Cache`: `HIT` or `MISS`.
- `X-Edge-ID`: configured Edge ID.
- `X-Edge-Location`: configured region.
- `Content-Type`: asset MIME type.

## Cache design

Each cached asset has two files:

``` text
edge/cache/india/
├── asset_<id>.body
└── asset_<id>.json
```

`.body` contains the binary bytes. `.json` stores metadata such as ID,
original filename, content type, version and cache creation time. A
neutral `.body` extension allows the same cache mechanism to handle
images, PDFs, videos and other file types.

### TTL

TTL is based on the original cache creation timestamp. An expired entry
is not served as a valid hit. Updating the access time does not reset
TTL.

### LRU

The metadata file modification time is used as the last-access marker.
Eviction prefers expired entries, then least-recently-used eligible
entries until the configured size limit is met.

The current eviction pass runs at startup and on cache writes; it is not
yet a continuously running background maintenance process. An asset
currently locked by another request may be skipped and reconsidered
later.

### Concurrency

Per-asset locks coordinate requests for the same asset and reduce
duplicate cache fills. Eviction attempts to avoid removing entries
currently in use.

## Testing

Use a real asset ID returned by the Origin:

``` powershell
$id = "asset_REPLACE_WITH_REAL_ID"
curl.exe -I "http://localhost:8081/assets/$id"
curl.exe -I "http://localhost:8081/assets/$id"
```

For a new asset, the expected sequence is `X-Cache: MISS`, then
`X-Cache: HIT`. Repeat against port `8082` to test the independent
Europe cache.

Inspect cache directories:

``` powershell
Get-ChildItem .\edge\cache\india
Get-ChildItem .\edge\cache\europe
```

For deletion testing: cache the asset in both Edges, delete it through
Origin, confirm both `.body` and `.json` files disappear, then request
it through both Edges and confirm the stale copy is not served.

For a short TTL/LRU development test, temporarily use a small policy
such as:

``` json
{
  "cache_ttl": "30s",
  "max_cache_bytes": 1048576
}
```

Restart the Edge after changing its config. Cache an asset, request it
again before expiry, wait beyond the TTL, and request again. To test
LRU, cache enough data to exceed the capacity and inspect which eligible
entries are removed.

Format/build/test:

``` powershell
gofmt -w .
go build .
go build .\edge
go test ./...
```

## Design decisions and limitations

- Go standard library HTTP server and primitives are used for a
  transparent learning experience.
- Origin metadata is persisted in JSON rather than SQLite.
- India and Europe are separate Edge processes using the same reusable
  implementation.
- Region selection is manual initially; IP geolocation and automatic
  routing are not implemented.
- The local setup is a development topology, not real geographic
  distribution.
- There is no update route or update workflow at this stage.
- Invalidation notification failures are logged but not durably retried.
