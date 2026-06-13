CREATE TABLE domain_events (
    id              INTEGER NOT NULL PRIMARY KEY,
    occurred_on     TEXT NOT NULL,
    name            TEXT NOT NULL,
    entity_id       TEXT NOT NULL,
    entity_version  INTEGER NOT NULL,
    payload         TEXT NOT NULL
);

CREATE INDEX domain_events_entity_id ON domain_events(entity_id);
CREATE UNIQUE INDEX domain_events_entity_version ON domain_events(entity_id, entity_version);

CREATE TABLE projections (
    name        TEXT NOT NULL PRIMARY KEY,
    last_event  INTEGER NOT NULL DEFAULT 0
);

