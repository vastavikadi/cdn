# WTF I Brainstormed --- How This CDN Project Evolved

> A project journal of the original idea

This document records the project as it developed through the planning
and implementation. It distinguishes the initial direction
from the architecture and features added during implementation. Future
work is not presented as already complete.

## 1. The original spark

The starting point was straightforward:

**Build a CDN in Go and connect to the Next.js frontend.**

The Next.js interface was intended to be kept as the place to upload assets, see uploaded assets,
choose a delivery location and request content.

The initial mental model:

- A Go backend stores uploaded files.
- The frontend lets a user choose where an asset should be delivered
  from.
- A CDN-like layer serves the file from the selected location.

## 2. The first architectural shape

The project evolved into a distinction between **Origin** and **Edge**.

### Origin

Origin is the source of truth. It receives uploads, stores original
files and metadata, lists assets, looks up assets by ID, serves original
binary content and deletes the original file and metadata.

### Edge

An Edge is a delivery/cache server. It checks its local cache, serves a
local copy on a hit, fetches from Origin on a miss and stores the
fetched body and metadata for later requests.

## 3. The architecture I settled on

Instead of one Go process pretending to be multiple regions, I chose
one reusable Edge implementation and run it as two independent
processes.

``` mermaid
flowchart TB
    Client["Existing Next.js frontend"]
    Origin["Go Origin :8080"]
    Store[("Original asset files")]
    Metadata[("data/assets.json")]
    India["India Edge process :8081"]
    Europe["Europe Edge process :8082"]
    IndiaCache[("India cache directory")]
    EuropeCache[("Europe cache directory")]

    Client -->|"Upload / list / delete"| Origin
    Origin --> Store
    Origin --> Metadata
    Client -->|"Choose India"| India
    Client -->|"Choose Europe"| Europe
    India -->|"MISS: fetch content"| Origin
    Europe -->|"MISS: fetch content"| Origin
    India --> IndiaCache
    Europe --> EuropeCache
```

Why this design?

- India and Europe are separate processes.
- Each has its own port, region identity and cache directory.
- Both run the same Edge code.
- Region-specific settings live in JSON configuration files.
- Adding another logical region can mean adding another config and
  running another instance.
- The frontend can choose a region manually without requiring IP
  geolocation or DNS-based routing.

The current ports are Origin `8080`, India Edge `8081`, Europe Edge
`8082`. These are local development ports, not public production
endpoints.

## 4. Storage: JSON metadata, not SQLite

The chosen Origin metadata store is a local JSON file. The asset model
contains fields such as ID, filename, storage name, content type, size,
version, checksum and upload time.

The original binary is stored separately under the assets directory;
metadata is stored in `data/assets.json`.

This keeps the first implementation easy to inspect. A database can be
considered later if the project needs stronger concurrent access,
indexing, multi-instance operation or advanced querying.

## 5. Asset identity and storage

Asset IDs have this form:

``` text
asset_<24 lowercase hexadecimal characters>
```

The ID is separate from the original filename. The original filename is
retained as metadata, while the stored filename is generated from the
asset ID and extension. The Origin calculates a SHA-256 checksum during
upload.

## 6. The first successful CDN delivery path

The first major milestone was getting an asset from Origin through an
Edge.

``` mermaid
sequenceDiagram
    participant Browser
    participant Edge as India Edge :8081
    participant Origin as Origin :8080
    participant Disk as Edge cache

    Browser->>Edge: GET /assets/{id}
    Edge->>Edge: Validate asset ID
    Edge->>Disk: Check local cache
    alt Cache hit
        Disk-->>Edge: Cached body and metadata
        Edge-->>Browser: Asset (X-Cache: HIT)
    else Cache miss
        Edge->>Origin: GET /api/assets/{id}/content
        Origin-->>Edge: Original bytes and headers
        Edge->>Disk: Store .body and .json
        Edge-->>Browser: Asset (X-Cache: MISS)
    end
```

A useful debugging moment happened here. The Edge returned
`400 Invalid asset ID` even though the ID was valid. The validator was
correct, but the condition in the handler was reversed:

``` go
if validAssetID(id) {
    // incorrectly reject a valid ID
}
```

It needed to be:

``` go
if !validAssetID(id) {
    // reject only an invalid ID
}
```

The first successful response included `200 OK`,
`Content-Type: image/png`, `X-Cache: MISS`, `X-Edge-Id: edge-india` and
`X-Edge-Location: India`.

## 7. Why the cache uses `.body` and `.json`

Each cached asset is represented by two files:

``` text
edge/cache/india/
├── asset_<id>.body
└── asset_<id>.json
```

The `.body` file stores binary bytes. The `.json` file stores metadata
required to serve the body correctly, including the original filename,
content type, version and cache creation time.

The neutral `.body` extension lets the same cache mechanism store
images, PDFs, videos and other file types.

## 8. Concurrency: one asset, one in-flight cache fill

The Edge uses per-asset locks. If several clients request the same
uncached asset simultaneously, requests coordinate around the cache
lookup/fill operation. This reduces duplicate Origin requests.

Eviction also needs to avoid deleting an asset while another request is
using it.

## 9. Cache invalidation: deletion must reach the Edges

A cache creates a consistency problem: Origin can delete an asset while
an Edge still has a cached copy. The current solution is an
Origin-to-Edge notification.

``` mermaid
sequenceDiagram
    participant UI as Next.js frontend
    participant Origin
    participant India as India Edge
    participant Europe as Europe Edge

    UI->>Origin: DELETE /api/assets/{id}
    Origin->>Origin: Delete original file and metadata
    Origin->>India: POST /internal/cache/{id}/invalidate
    Origin->>Europe: POST /internal/cache/{id}/invalidate
    India->>India: Remove cached .body and .json
    Europe->>Europe: Remove cached .body and .json
    Origin-->>UI: Deletion result
```

The Edge invalidation endpoint requires an `X-Edge-Token` header.

Current behavior: - Origin deletes the original file and metadata. -
Origin attempts to notify both Edges. - Each Edge removes the
corresponding `.body` and `.json`. - If an Edge is unavailable, Origin
logs the notification failure.

Durable retry, reconciliation or another recovery mechanism for missed
notifications has not yet been implemented. This is an important
reliability task before production.

There is **no update route or update workflow**. Invalidation currently
supports deletion only.

## 10. Cache lifetime and capacity: TTL + LRU

After caching and deletion invalidation worked, I added cache
management.

### TTL --- Time To Live

TTL controls how long a cached entry is considered fresh. The
configuration accepts Go duration strings such as `24h`. The original
cache creation timestamp is stored in metadata; updating access time
does not reset TTL.

### LRU --- Least Recently Used

LRU helps keep the cache within its configured capacity.

- The metadata file modification time is used as the last-access marker.
- Expired entries are preferred for removal.
- If the cache remains over capacity, least-recently-used eligible
  entries are removed.

Each Edge has its own TTL and maximum cache size. The initial example
used 24 hours and 500 MiB.

The current eviction pass runs at startup and on cache writes; it is not
yet a continuously running background maintenance process. An asset
currently locked by another request may be skipped and reconsidered
later.