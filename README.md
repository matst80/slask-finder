# slask-finder

`go run . -profiling`

`brew install graphviz`

`go tool pprof "http://localhost:8080/debug/pprof/profile?seconds=5"`

`go tool pprof "http://localhost:8080/debug/pprof/heap"`

`go tool pprof "http://localhost:8080/debug/pprof/allocs"`

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)

## Installation

Instructions on how to install and set up your project.

## Usage

Instructions on how to use your project, including examples and screenshots if applicable.

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
