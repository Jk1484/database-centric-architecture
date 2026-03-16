# sqlm Syntax

sqlm is a module system for PostgreSQL SQL. It adds package declaration, imports, and init functions on top of raw SQL. The compiler owns exactly three things — everything else is passed through as-is.

---

## File Structure

```sqlm
package users

import "queries"

func init() {
    DROP FUNCTION IF EXISTS users.create(users.create_request);
    DROP TYPE IF EXISTS users.create_request;
    DROP TYPE IF EXISTS users.create_response;
}

CREATE TYPE users.create_request AS (
    name  TEXT,
    email TEXT
);

CREATE TYPE users.create_response AS (
    id         UUID,
    name       TEXT,
    email      TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

CREATE OR REPLACE FUNCTION users.create(req users.create_request)
RETURNS users.create_response
SECURITY DEFINER
AS $$
DECLARE
    result users.create_response;
BEGIN
    INSERT INTO public.users (name, email)
    VALUES (req.name, req.email)
    RETURNING id, name, email, created_at, updated_at
    INTO result;

    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

---

## What the compiler owns

### 1. `package`

Declares the package. Compiles to `CREATE SCHEMA IF NOT EXISTS <name>`.

```sqlm
package users
```

### 2. `import`

Imports another package by name. The compiler resolves it to a directory and ensures it is executed first.

```sqlm
import "queries"
import "reports"
```

### 3. `func init()`

Runs before all other SQL in the file. No parameters, no return type. Use it for drops or any setup that must happen before the file's SQL runs.

```sqlm
func init() {
    DROP FUNCTION IF EXISTS users.create(users.create_request);
    DROP TYPE IF EXISTS users.create_request;
    DROP TYPE IF EXISTS users.create_response;
}
```

If absent, nothing runs before the file's SQL — no auto-drops, no magic.

---

## Everything else is raw SQL

Types, functions, triggers, views — written in plain PostgreSQL SQL, passed through by the compiler unchanged. Full PostgreSQL syntax is available with no restrictions.

---

## Compiled output order

For a given package, the compiler outputs:

1. `CREATE SCHEMA IF NOT EXISTS <package>`
2. `func init()` bodies — in file order
3. remaining SQL — in file order

Imported packages are output before the importing package, recursively.

---

## Comments

```sqlm
-- this is a comment
```

---

## Full compiled example

Input:

```sqlm
package users

import "queries"

func init() {
    DROP FUNCTION IF EXISTS users.create(users.create_request);
    DROP TYPE IF EXISTS users.create_request;
    DROP TYPE IF EXISTS users.create_response;
}

CREATE TYPE users.create_request AS (
    name  TEXT,
    email TEXT
);

CREATE OR REPLACE FUNCTION users.create(req users.create_request)
RETURNS users.create_response AS $$
BEGIN
    ...
END;
$$ LANGUAGE plpgsql;
```

Output:

```sql
CREATE SCHEMA IF NOT EXISTS queries;

-- queries package SQL here

CREATE SCHEMA IF NOT EXISTS users;

DROP FUNCTION IF EXISTS users.create(users.create_request);
DROP TYPE IF EXISTS users.create_request;
DROP TYPE IF EXISTS users.create_response;

CREATE TYPE users.create_request AS (
    name  TEXT,
    email TEXT
);

CREATE OR REPLACE FUNCTION users.create(req users.create_request)
RETURNS users.create_response AS $$
BEGIN
    ...
END;
$$ LANGUAGE plpgsql;
```
