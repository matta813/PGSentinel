ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'administrator'
    CHECK (role IN ('administrator', 'operator', 'viewer'));

CREATE TRIGGER users_bounded_insert
BEFORE INSERT ON users
WHEN (SELECT COUNT(*) FROM users) >= 100
BEGIN
    SELECT RAISE(ABORT, 'local user limit reached');
END;
