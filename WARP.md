# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

Slask-finder is a Go-based e-commerce search system built with microservices architecture. It provides sophisticated search, faceting, and recommendation capabilities with AI-powered embeddings for semantic search.

## Architecture

### Core Services (4 microservices)

1. **Reader Service** (`cmd/reader/`) - Read-only search and retrieval API
   - Handles search queries, faceting, item retrieval, suggestions
   - Serves HTTP API on port 8080
   - Loads data from disk storage and maintains in-memory indexes

2. **Writer Service** (`cmd/writer/`) - Administrative write operations
   - Manages facets, settings, field configurations 
   - Google OAuth authentication for admin operations
   - RabbitMQ integration for data updates

3. **Embeddings Service** (`cmd/embeddings/`) - AI-powered semantic search
   - Uses Ollama for generating embeddings with configurable models
   - Supports multiple Ollama endpoints with round-robin load balancing  
   - Listens to RabbitMQ for new items to embed

4. **Price Watcher Service** (`cmd/pricewatcher/`) - Price monitoring
   - Tracks price changes for items

### Key Packages

- `pkg/index/` - In-memory item indexing with stock management
- `pkg/search/` - Free-text search with tokenization and trie-based indexing
- `pkg/facet/` - Faceted search with various field types (key, integer, number buckets)
- `pkg/embeddings/` - Ollama integration and embeddings management
- `pkg/storage/` - Disk persistence layer with gzip compression
- `pkg/common/` - Shared utilities including graceful shutdown, HTTP helpers, session management
- `pkg/types/` - Core data structures and business logic

### Data Flow

- Items flow through RabbitMQ topics: `item_added`, `item_changed`, `item_deleted`
- Data persists to disk in `data/` directory with country-specific organization
- Reader service loads full dataset into memory for fast query processing
- Embeddings service processes items asynchronously and stores vector embeddings

## Development Commands

### Build & Run

```bash
# Run with profiling enabled  
go run . -profiling

# Build individual services
go build -o slask-reader ./cmd/reader
go build -o slask-writer ./cmd/writer  
go build -o slask-embeddings ./cmd/embeddings
go build -o price-watcher ./cmd/pricewatcher
```

### Docker

```bash
# Build reader service
docker build -f cmd/reader/Dockerfile -t slask-reader .

# Build writer service
docker build -f cmd/writer/Dockerfile -t slask-writer .

# Main Dockerfile builds all services in multi-stage build
docker build -t slask-finder .
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./pkg/embeddings/
go test ./pkg/facet/
go test ./pkg/search/

# Run single test
go test -run TestSpecificFunction ./pkg/package/
```

### Linting

The project uses golangci-lint with comprehensive linter configuration (`.golangci.yml`):

```bash
# Install linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run full lint
golangci-lint run ./...

# Quick lint (new files only) 
golangci-lint run --new-from-rev=origin/main
```

**Git pre-commit hooks** are available in `.githooks/` - enable with:
```bash
git config core.hooksPath .githooks
```

### Profiling

```bash
# Install graphviz for profile visualization
brew install graphviz  # macOS
# or sudo pacman -S graphviz  # Arch Linux

# Profile running service (reader runs on :8081 when profiling enabled)
go tool pprof "http://localhost:8081/debug/pprof/profile?seconds=5"
go tool pprof "http://localhost:8081/debug/pprof/heap"
go tool pprof "http://localhost:8081/debug/pprof/allocs"

# Profile startup
go run . -profile-startup=startup.prof
go tool pprof startup.prof
```

## Environment Configuration

### Required Environment Variables

- `COUNTRY` - Country code (default: "se")
- `RABBIT_HOST` - RabbitMQ connection URL (required for writer/embeddings services)

### Ollama Embeddings Configuration  

- `OLLAMA_MODEL` - Model name (default: "elkjop-ecom")
- `OLLAMA_URL` - Ollama API endpoint (default: "http://10.10.11.135:11434/api/embeddings")

### Authentication (Writer Service)

Uses Google OAuth for admin authentication. Configure through environment or falls back to mock auth for development.

## Database Schemas

### ClickHouse Analytics Tables

```sql
-- User session tracking
CREATE TABLE user_session (
    session_id UInt32,
    timestamp DateTime DEFAULT now(),
    language TEXT,
    user_agent TEXT,
    ip TEXT
) ENGINE = MergeTree ORDER BY timestamp;

-- User action tracking  
CREATE TABLE user_action (
    session_id UInt32,
    evt UInt32,
    item_id UInt64,
    metric Float32,
    timestamp DateTime DEFAULT now()
) ENGINE = MergeTree ORDER BY timestamp;

-- Search query tracking
CREATE TABLE user_search (
    session_id UInt32,
    evt UInt32, 
    query TEXT,
    facets Map(UInt64,String),
    timestamp DateTime DEFAULT now()
) ENGINE = MergeTree ORDER BY timestamp;
```

## Key Design Patterns

### Graceful Shutdown
All services use `pkg/common/graceful.go` for proper shutdown handling with configurable timeouts and cleanup hooks.

### Error Handling
Follow Go error handling conventions. The linter enforces error checking (`errcheck` enabled).

### Memory Management
Reader service loads full dataset in memory for performance. Use `SaveTrigger` API to persist changes before shutdown.

### Concurrent Safety
Services use proper synchronization (e.g., `sync.RWMutex` in writer service) for shared data structures.

### Load Balancing
Embeddings service supports multiple Ollama endpoints with automatic round-robin distribution for scalability.

## CI/CD

GitHub Actions workflow (`.github/workflows/docker-image.yml`) automatically builds and pushes Docker images to Docker Hub on pushes to main/feature branches.

Tagged as `matst80/s10n:latest` in the registry.