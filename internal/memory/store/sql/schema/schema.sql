-- Memory Store Schema
-- This file contains the SQLite schema for the memory and history tables.

-- ============================================================================
-- MEMORIES TABLE
-- Stores user preferences, facts, and extracted information with embeddings
-- ============================================================================

CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    text TEXT NOT NULL,
    tags TEXT,
    source TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    confidence REAL NOT NULL,
    stability_days REAL NOT NULL,
    last_retrieved_at INTEGER,
    provider TEXT NOT NULL,
    model_id TEXT NOT NULL,
    dim INTEGER NOT NULL,
    embedding BLOB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at);

-- ============================================================================
-- HISTORY TABLE
-- Stores conversation turns for context retrieval
-- ============================================================================

CREATE TABLE IF NOT EXISTS history (
    id TEXT PRIMARY KEY,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    session_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_history_created_at ON history(created_at);
CREATE INDEX IF NOT EXISTS idx_history_session ON history(session_id);

-- ============================================================================
-- HISTORY FTS5 (Full-Text Search)
-- Virtual table for fast text search on history content
-- ============================================================================

CREATE VIRTUAL TABLE IF NOT EXISTS history_fts USING fts5(
    content,
    content='history',
    content_rowid='rowid'
);

-- Triggers to keep FTS index in sync with history table
CREATE TRIGGER IF NOT EXISTS history_ai AFTER INSERT ON history BEGIN
    INSERT INTO history_fts(rowid, content) VALUES (NEW.rowid, NEW.content);
END;

CREATE TRIGGER IF NOT EXISTS history_ad AFTER DELETE ON history BEGIN
    INSERT INTO history_fts(history_fts, rowid, content) VALUES('delete', OLD.rowid, OLD.content);
END;

CREATE TRIGGER IF NOT EXISTS history_au AFTER UPDATE ON history BEGIN
    INSERT INTO history_fts(history_fts, rowid, content) VALUES('delete', OLD.rowid, OLD.content);
    INSERT INTO history_fts(rowid, content) VALUES (NEW.rowid, NEW.content);
END;

-- ============================================================================
-- MEMORIES FTS5 (Full-Text Search)
-- Virtual table for fast text search on memory text
-- ============================================================================

CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    text,
    content='memories',
    content_rowid='rowid'
);

-- Triggers to keep FTS index in sync with memories table
CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(rowid, text) VALUES (NEW.rowid, NEW.text);
END;

CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, text) VALUES('delete', OLD.rowid, OLD.text);
END;

CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, text) VALUES('delete', OLD.rowid, OLD.text);
    INSERT INTO memories_fts(rowid, text) VALUES (NEW.rowid, NEW.text);
END;
