BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    name       TEXT NOT NULL CHECK (name <> ''),
    surname    TEXT NOT NULL CHECK (surname <> ''),
    patronymic TEXT          CHECK (patronymic <> ''),

    address TEXT NOT NULL CHECK (address <> '')
);

CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    sess_begin TIMESTAMPTZ NOT NULL,
    sess_end   TIMESTAMPTZ,

    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    task_id INTEGER NOT NULL REFERENCES tasks (id) ON DELETE CASCADE
);

COMMIT;