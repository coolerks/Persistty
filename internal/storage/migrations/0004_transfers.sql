CREATE TABLE uploads (
 id TEXT NOT NULL PRIMARY KEY,
 project_id TEXT NOT NULL,
 folder_id TEXT NOT NULL,
 project_version INTEGER NOT NULL,
 relative_path TEXT NOT NULL,
 batch_id TEXT NOT NULL,
 size INTEGER NOT NULL CHECK(size>=0),
 sha256 TEXT NOT NULL,
 chunk_bytes INTEGER NOT NULL CHECK(chunk_bytes>0),
 expected_version_json TEXT,
 created_at TEXT NOT NULL,
 expires_at TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('pending','completed','skipped','failed')),
 result_json TEXT
);
CREATE INDEX uploads_batch_idx ON uploads(batch_id,expires_at);
CREATE TABLE upload_chunks (
 upload_id TEXT NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
 chunk_index INTEGER NOT NULL CHECK(chunk_index>=0),
 sha256 TEXT NOT NULL,
 size INTEGER NOT NULL CHECK(size>=0),
 PRIMARY KEY(upload_id,chunk_index)
);
CREATE TABLE archives (
 id TEXT NOT NULL PRIMARY KEY,
 project_id TEXT NOT NULL,
 folder_id TEXT NOT NULL,
 project_version INTEGER NOT NULL,
 relative_path TEXT NOT NULL,
 created_at TEXT NOT NULL,
 expires_at TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('pending','ready','failed','cancelled')),
 size INTEGER NOT NULL DEFAULT 0,
 error_code TEXT NOT NULL DEFAULT ''
);
