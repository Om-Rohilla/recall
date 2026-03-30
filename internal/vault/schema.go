package vault

import (
	"database/sql"
	"fmt"
)

const schemaVersion = 1

const createSchema = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
PRAGMA secure_delete = ON;
PRAGMA auto_vacuum = INCREMENTAL;

CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS commands (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    raw           TEXT NOT NULL UNIQUE,
    binary_name   TEXT NOT NULL,
    subcommand    TEXT DEFAULT '',
    flags         TEXT DEFAULT '[]',
    category      TEXT DEFAULT '',
    frequency     INTEGER DEFAULT 1,
    first_seen    TEXT NOT NULL,
    last_seen     TEXT NOT NULL,
    last_exit     INTEGER,
    avg_duration  REAL
);

CREATE TABLE IF NOT EXISTS contexts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    command_id    INTEGER NOT NULL REFERENCES commands(id) ON DELETE CASCADE,
    cwd           TEXT DEFAULT '',
    git_repo      TEXT DEFAULT '',
    git_branch    TEXT DEFAULT '',
    project_type  TEXT DEFAULT '',
    timestamp     TEXT NOT NULL,
    exit_code     INTEGER,
    duration_ms   INTEGER,
    session_id    TEXT DEFAULT ''
);

CREATE TABLE IF NOT EXISTS knowledge (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    command       TEXT NOT NULL,
    description   TEXT NOT NULL,
    intents       TEXT NOT NULL DEFAULT '[]',
    category      TEXT NOT NULL DEFAULT '',
    flags_doc     TEXT DEFAULT '{}',
    examples      TEXT DEFAULT '[]',
    danger_level  TEXT DEFAULT 'safe'
);

CREATE TABLE IF NOT EXISTS patterns (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    template        TEXT NOT NULL UNIQUE,
    frequency       INTEGER DEFAULT 1,
    suggested_alias TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_commands_binary ON commands(binary_name);
CREATE INDEX IF NOT EXISTS idx_commands_category ON commands(category);
CREATE INDEX IF NOT EXISTS idx_commands_last_seen ON commands(last_seen);
CREATE INDEX IF NOT EXISTS idx_commands_frequency ON commands(frequency);
CREATE INDEX IF NOT EXISTS idx_contexts_command_id ON contexts(command_id);
CREATE INDEX IF NOT EXISTS idx_contexts_timestamp ON contexts(timestamp);
CREATE INDEX IF NOT EXISTS idx_knowledge_category ON knowledge(category);
`

const createFTS = `
CREATE VIRTUAL TABLE IF NOT EXISTS commands_fts USING fts5(
    raw,
    binary_name,
    subcommand,
    category,
    content='commands',
    content_rowid='id'
);
`

const createFTSTriggers = `
CREATE TRIGGER IF NOT EXISTS commands_ai AFTER INSERT ON commands BEGIN
    INSERT INTO commands_fts(rowid, raw, binary_name, subcommand, category)
    VALUES (new.id, new.raw, new.binary_name, new.subcommand, new.category);
END;

CREATE TRIGGER IF NOT EXISTS commands_ad AFTER DELETE ON commands BEGIN
    INSERT INTO commands_fts(commands_fts, rowid, raw, binary_name, subcommand, category)
    VALUES ('delete', old.id, old.raw, old.binary_name, old.subcommand, old.category);
END;

CREATE TRIGGER IF NOT EXISTS commands_au AFTER UPDATE ON commands BEGIN
    INSERT INTO commands_fts(commands_fts, rowid, raw, binary_name, subcommand, category)
    VALUES ('delete', old.id, old.raw, old.binary_name, old.subcommand, old.category);
    INSERT INTO commands_fts(rowid, raw, binary_name, subcommand, category)
    VALUES (new.id, new.raw, new.binary_name, new.subcommand, new.category);
END;
`

const createKnowledgeFTS = `
CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_fts USING fts5(
    command,
    description,
    intents,
    category,
    content='knowledge',
    content_rowid='id'
);
`

const createKnowledgeFTSTriggers = `
CREATE TRIGGER IF NOT EXISTS knowledge_ai AFTER INSERT ON knowledge BEGIN
    INSERT INTO knowledge_fts(rowid, command, description, intents, category)
    VALUES (new.id, new.command, new.description, new.intents, new.category);
END;

CREATE TRIGGER IF NOT EXISTS knowledge_ad AFTER DELETE ON knowledge BEGIN
    INSERT INTO knowledge_fts(knowledge_fts, rowid, command, description, intents, category)
    VALUES ('delete', old.id, old.command, old.description, old.intents, old.category);
END;

CREATE TRIGGER IF NOT EXISTS knowledge_au AFTER UPDATE ON knowledge BEGIN
    INSERT INTO knowledge_fts(knowledge_fts, rowid, command, description, intents, category)
    VALUES ('delete', old.id, old.command, old.description, old.intents, old.category);
    INSERT INTO knowledge_fts(rowid, command, description, intents, category)
    VALUES (new.id, new.command, new.description, new.intents, new.category);
END;
`

type migration struct {
	version int
	sql     string
}

var migrations = []migration{
	{
		version: 2,
		sql:     `PRAGMA secure_delete = ON; PRAGMA auto_vacuum = INCREMENTAL;`,
	},
}

func initSchema(db *sql.DB) error {
	if _, err := db.Exec(createSchema); err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}

	if _, err := db.Exec(createFTS); err != nil {
		return fmt.Errorf("creating FTS5 index: %w", err)
	}

	if _, err := db.Exec(createFTSTriggers); err != nil {
		return fmt.Errorf("creating FTS5 triggers: %w", err)
	}

	if _, err := db.Exec(createKnowledgeFTS); err != nil {
		return fmt.Errorf("creating knowledge FTS5: %w", err)
	}

	if _, err := db.Exec(createKnowledgeFTSTriggers); err != nil {
		return fmt.Errorf("creating knowledge FTS5 triggers: %w", err)
	}

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count)
	if err != nil {
		return fmt.Errorf("checking schema version: %w", err)
	}
	if count == 0 {
		if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (?)", schemaVersion); err != nil {
			return fmt.Errorf("inserting schema version: %w", err)
		}
	}

	if err := runMigrations(db); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}

func runMigrations(db *sql.DB) error {
	var currentVersion int
	if err := db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&currentVersion); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	for _, m := range migrations {
		if m.version <= currentVersion {
			continue
		}
		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("migration v%d: %w", m.version, err)
		}
		if _, err := db.Exec("UPDATE schema_version SET version = ?", m.version); err != nil {
			return fmt.Errorf("updating schema version to %d: %w", m.version, err)
		}
	}

	return nil
}
