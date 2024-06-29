DROP TABLE IF EXISTS user_action
CREATE TABLE IF NOT EXISTS user_action
(
    session_id UInt32,
    evt UInt16,
    timestamp DateTime,
		item_id UInt64,
    metric Float32
)
ENGINE = MergeTree
PRIMARY KEY (session_id, timestamp, evt)

DROP TABLE IF EXISTS user_session
CREATE TABLE IF NOT EXISTS user_session
(
    session_id UInt32,
    timestamp DateTime,
		language String,
    user_agent String,
    ip String,
)
ENGINE = MergeTree
PRIMARY KEY (session_id, timestamp)

DROP TABLE IF EXISTS user_search
CREATE TABLE IF NOT EXISTS user_search
(
    session_id UInt32,
    evt UInt16,
    timestamp DateTime,
		query String,
		facets Map(UInt64,String),
)
ENGINE = MergeTree
PRIMARY KEY (session_id, timestamp, evt)

