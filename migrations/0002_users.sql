CREATE TABLE users (
    id INTEGER NOT NULL PRIMARY KEY,
    email    TEXT NOT NULL UNIQUE,
    salt     TEXT NOT NULL,
    password TEXT NOT NULL,
    roles    TEXT NOT NULL
);

CREATE TABLE sessions (
    id      TEXT NOT NULL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    ip      TEXT NOT NULL,
    agent   TEXT NOT NULL,
    expires TEXT NOT NULL
);
