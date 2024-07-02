BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    name       TEXT NOT NULL CHECK (name <> ''),
    surname    TEXT NOT NULL CHECK (surname <> ''),
    patronymic TEXT          CHECK (patronymic <> ''),

    passport_serie  INTEGER NOT NULL CHECK (passport_serie > 0),
    passport_number INTEGER NOT NULL CHECK (passport_number > 0),

    address TEXT NOT NULL CHECK (address <> ''),

    CONSTRAINT unique_passport UNIQUE (passport_serie, passport_number)
);

CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    sess_begin TIMESTAMPTZ NOT NULL,
    sess_end   TIMESTAMPTZ,

    task_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE
);

COMMIT;