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

### Using Ollama Embeddings

The project now supports generating embeddings using Ollama's HTTP API with the "mxbai-embed-large" model. This enables more accurate semantic search and item similarity matching.

To use the Ollama embeddings engine:

```go
import "github.com/matst80/slask-finder/pkg/embeddings"

// Create a new Ollama embeddings engine with default configuration
// (uses "mxbai-embed-large" model and http://10.10.10.100:11434/api/embeddings endpoint)
embeddingsEngine := embeddings.NewOllamaEmbeddingsEngine()

// Or create with custom configuration
customEngine := embeddings.NewOllamaEmbeddingsEngineWithConfig(
    "nomic-embed-text", 
    "http://custom-endpoint:11434/api/embeddings",
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

Make sure the Ollama server is running and accessible at the configured endpoint.

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
