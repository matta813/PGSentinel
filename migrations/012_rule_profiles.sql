CREATE TABLE rule_profiles (id TEXT PRIMARY KEY,name TEXT NOT NULL UNIQUE,description TEXT NOT NULL DEFAULT '',entries_json TEXT NOT NULL,created_at TEXT NOT NULL,updated_at TEXT NOT NULL);
