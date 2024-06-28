DROP TABLE IF EXISTS user_action;
CREATE TABLE IF NOT EXISTS user_action
(
    session_id UInt32,
    evt UInt16,
    timestamp DateTime,
		item_id UInt64,
		query String,
		facets Map(UInt64,String),
    metric Float32
)
ENGINE = MergeTree
PRIMARY KEY (session_id, timestamp, evt);


INSERT INTO user_action (session_id, evt, timestamp, item_id, metric) VALUES
    (101, 1,                                  now(),  21841,     1.0    ),
    (101, 1,                                  now(),  21841,     1.0    ),
		(102, 1,                                  now(),  21841,     1.0    ),
		(102, 1,                                  now(),  21841,     1.0    ),
		(101, 2,                                  now(),  21841,     10.0    );

INSERT INTO user_action (session_id, evt, timestamp, item_id, query, metric) VALUES
    (101, 1,                                  now(),  21841,     'test data', 1.0    );
    