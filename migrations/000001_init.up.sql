BEGIN;

CREATE SCHEMA todoapp;

CREATE TABLE todoapp.users (
    id UUID PRIMARY KEY,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    full_name VARCHAR(100) NOT NULL CHECK (char_length(full_name) BETWEEN 3 AND 100)
);

CREATE TABLE todoapp.tasks (
    id UUID PRIMARY KEY,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    title VARCHAR(100) NOT NULL CHECK (char_length(title) BETWEEN 1 AND 100),
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    author_user_id UUID NOT NULL REFERENCES todoapp.users(id),
    CONSTRAINT tasks_completion_consistent CHECK (
        (completed AND completed_at IS NOT NULL)
        OR (NOT completed AND completed_at IS NULL)
    ),
    CONSTRAINT tasks_completion_after_creation CHECK (completed_at >= created_at)
);

COMMIT;
