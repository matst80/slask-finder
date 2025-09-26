# slask-finder

`go run . -profiling`

`brew install graphviz`

`go tool pprof "http://localhost:8081/debug/pprof/profile?seconds=5"`

`go tool pprof "http://localhost:8081/debug/pprof/heap"`

`go tool pprof "http://localhost:8081/debug/pprof/allocs"`

`-profile-startup=startup.prof`

`go tool pprof startup.prof`

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)

## Installation

Instructions on how to install and set up your project.

## Usage

Instructions on how to use your project, including examples and screenshots if applicable.

# For reader service
docker build -f cmd/reader/Dockerfile -t slask-reader .

# For writer service  
docker build -f cmd/writer/Dockerfile -t slask-writer .

### Using Ollama Embeddings

The project now supports generating embeddings using Ollama's HTTP API with the "mxbai-embed-large" model. This enables more accurate semantic search and item similarity matching.

To use the Ollama embeddings engine:

```go
import "github.com/matst80/slask-finder/pkg/embeddings"

// Create a new Ollama embeddings engine with default configuration
// (uses "mxbai-embed-large" model and http://10.10.10.100:11434/api/embeddings endpoint)
embeddingsEngine := embeddings.NewOllamaEmbeddingsEngine()

// Or create with custom configuration for a single endpoint
customEngine := embeddings.NewOllamaEmbeddingsEngineWithConfig(
    "nomic-embed-text", 
    "http://custom-endpoint:11434/api/embeddings",
)

// Use multiple endpoints with round-robin load balancing
multiEndpointEngine := embeddings.NewOllamaEmbeddingsEngineWithMultipleEndpoints(
    "nomic-embed-text",
    []string{
        "http://server1:11434/api/embeddings",
        "http://server2:11434/api/embeddings",
        "http://server3:11434/api/embeddings",
    },
)

// Generate embeddings for text
embeddings, err := embeddingsEngine.GenerateEmbeddings("text to generate embeddings for")
if err != nil {
    // Handle error
}

// Generate embeddings from an item
itemEmbeddings, err := embeddingsEngine.GenerateEmbeddingsFromItem(item)
if err != nil {
    // Handle error
}
```

### Multiple Endpoints and Load Balancing

When using `NewOllamaEmbeddingsEngineWithMultipleEndpoints`, requests are distributed across the provided endpoints using a round-robin algorithm. This provides several benefits:

1. **Load distribution**: Processing is spread across multiple Ollama servers, preventing any single server from becoming a bottleneck
2. **Fault tolerance**: If one server fails, requests will continue to be processed by the remaining servers
3. **Scalability**: You can add more servers to handle increased load as needed

The engine automatically handles the distribution of requests in a thread-safe manner, ensuring even load across all endpoints.

Make sure all Ollama servers are running and accessible at the configured endpoints.

## Contributing

Guidelines for contributing to your project, including how to report issues and submit pull requests.

## License

Information about the license under which your project is distributed.

## RabbitMQ

Exchanges topics

item_added
item_changed
item_deleted

## Clickhouse database

CREATE TABLE IF NOT EXISTS user_session
(
    session_id UInt32,
		timestamp DateTime DEFAULT now(),
		language TEXT,
		user_agent TEXT,
		ip TEXT
) ENGINE = MergeTree
ORDER BY timestamp;

CREATE TABLE IF NOT EXISTS user_action
(
    session_id UInt32,
		evt UInt32,
		item_id UInt64,
		metric Float32,
		timestamp DateTime DEFAULT now()
) ENGINE = MergeTree
ORDER BY timestamp;


CREATE TABLE IF NOT EXISTS user_search
(
    session_id UInt32,
		evt UInt32,
		query TEXT,
		facets Map(UInt64,String),
		timestamp DateTime DEFAULT now()
) ENGINE = MergeTree
ORDER BY timestamp;


LAST SYNC 1727297180740
