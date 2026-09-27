CREATE TABLE auth_config (id INTEGER PRIMARY KEY CHECK (id=1), password_fingerprint TEXT NOT NULL);
CREATE TABLE sessions (token_hash TEXT PRIMARY KEY, expires_at TEXT NOT NULL);
CREATE TABLE projects (
 id TEXT PRIMARY KEY CHECK (length(id)>0), name TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 200),
 version INTEGER NOT NULL CHECK (version BETWEEN 1 AND 9007199254740991), main_folder_id TEXT NOT NULL,
 FOREIGN KEY(id,main_folder_id) REFERENCES folders(project_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE folders (
 id TEXT PRIMARY KEY CHECK (length(id)>0), project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
 path TEXT NOT NULL CHECK (length(path)>0), UNIQUE(project_id,id), UNIQUE(project_id,path)
);
CREATE TABLE terminals (
 id TEXT PRIMARY KEY CHECK (length(id)>0), tmux_session_name TEXT NOT NULL UNIQUE,
 display_name TEXT NOT NULL, project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
 working_directory TEXT NOT NULL, created_at TEXT NOT NULL
);
