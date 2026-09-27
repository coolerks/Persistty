CREATE TABLE projects_new (
 id TEXT NOT NULL PRIMARY KEY CHECK (length(id)>0), name TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 200),
 version INTEGER NOT NULL CHECK (version BETWEEN 1 AND 9007199254740991), main_folder_id TEXT NOT NULL,
 FOREIGN KEY(id,main_folder_id) REFERENCES folders_new(project_id,id) DEFERRABLE INITIALLY DEFERRED
);
CREATE TABLE folders_new (
 id TEXT NOT NULL PRIMARY KEY CHECK (length(id)>0), project_id TEXT NOT NULL REFERENCES projects_new(id) ON DELETE CASCADE,
 path TEXT NOT NULL CHECK (length(path)>0), UNIQUE(project_id,id), UNIQUE(project_id,path)
);
CREATE TABLE terminals_new (
 id TEXT NOT NULL PRIMARY KEY CHECK (length(id)>0), tmux_session_name TEXT NOT NULL UNIQUE,
 display_name TEXT NOT NULL, project_id TEXT REFERENCES projects_new(id) ON DELETE SET NULL,
 working_directory TEXT NOT NULL, created_at TEXT NOT NULL
);
INSERT INTO projects_new SELECT * FROM projects;
INSERT INTO folders_new SELECT * FROM folders;
INSERT INTO terminals_new SELECT * FROM terminals;
DROP TABLE terminals;
DROP TABLE folders;
DROP TABLE projects;
ALTER TABLE projects_new RENAME TO projects;
ALTER TABLE folders_new RENAME TO folders;
ALTER TABLE terminals_new RENAME TO terminals;
