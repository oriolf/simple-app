CREATE TABLE unique_member_dnis (
    id              TEXT NOT NULL PRIMARY KEY,
    nif             VARCHAR(9) NOT NULL UNIQUE
);

CREATE TABLE members_table (
    id              TEXT NOT NULL PRIMARY KEY,
    projected_on    TEXT NOT NULL,
    nif             VARCHAR(9) NOT NULL,
    name            TEXT NOT NULL,
    joined_on       VARCHAR(10) NOT NULL,
    left_on         VARCHAR(10)
);

