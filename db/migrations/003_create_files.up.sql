CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    format TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    duration_seconds REAL,
    storage_key TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT files_format_check CHECK (format IN ('mp3', 'aac', 'wav', 'aiff'))
);

CREATE INDEX files_owner_id_idx ON files(owner_id);
