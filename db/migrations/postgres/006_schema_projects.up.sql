CREATE TABLE IF NOT EXISTS schema_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    publish_token_id UUID NOT NULL REFERENCES publish_tokens(id) ON DELETE RESTRICT,
    publish_token TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schema_projects_status_check CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT schema_projects_user_slug_unique UNIQUE (user_id, slug)
);

CREATE INDEX IF NOT EXISTS schema_projects_user_idx
    ON schema_projects (user_id, updated_at DESC);
