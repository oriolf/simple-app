CREATE TABLE members (
    id        INTEGER NOT NULL PRIMARY KEY,
    nif       VARCHAR(9)  UNIQUE NOT NULL,
    name      TEXT NOT NULL,
    joined_on VARCHAR(10) NOT NULL,
    left_on   VARCHAR(10),
    iban      VARCHAR(20)
);
