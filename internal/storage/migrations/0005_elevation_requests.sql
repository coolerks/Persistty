CREATE TABLE elevation_requests (
 id TEXT NOT NULL PRIMARY KEY,
 session_hash TEXT NOT NULL,
 payload_json TEXT NOT NULL,
 expires_at TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('prepared','executing','applied','rejected','cancelled','expired','indeterminate')),
 code TEXT,
 version_json TEXT,
 created_at TEXT NOT NULL
);
CREATE INDEX elevation_session_idx ON elevation_requests(session_hash,expires_at);
