-- +goose Up
CREATE TABLE IF NOT EXISTS notes (
  id INTEGER PRIMARY KEY,
  body TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_notes_created_at ON notes(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_notes_created_at;
DROP TABLE IF EXISTS notes;
