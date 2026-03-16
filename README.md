# Database-Centric Architecture

Business logic lives in the database — stored procedures, types, views, triggers. The Go layer is a thin transport: HTTP → call DB function → return result.

---

## Migrations

Two kinds of migrations:

### Schema
Regular, one-time migrations. Tables, indexes, sequences. Tracked and never re-run.

```
migrations/schema/
  001_users.sql
  002_posts.sql
```

### Entities
Types and functions. Dropped and recreated on every startup. Never tracked.

```
migrations/entities/
  main.sql
  queries/
    init.sql
    build_where_clause.sql
  users/
    init.sql
    create.sql
    get.sql
    update.sql
    delete.sql
    list.sql
```

---

## Include System

Entities use an include system modeled after Go's import mechanism.

### Rules

- `main.sql` is the single entry point. The runner only reads `main.sql` and follows its includes.
- Every directory exposes itself through `init.sql`, which includes its files.
- A file not reachable from `main.sql` is not run.
- A file included more than once is only run once (deduplication).
- Include order matters — a file must be included before anything that depends on it.

### Syntax

```sql
-- @include ./queries/init.sql
-- @include ./users/init.sql
```

### Example

```
-- main.sql
-- @include ./queries/init.sql   <- loads queries.build_where_clause, now in scope
-- @include ./users/init.sql     <- users functions can reference queries.*
```

```
-- queries/init.sql
-- @include ./build_where_clause.sql
```

```
-- users/init.sql
-- @include ./create.sql
-- @include ./get.sql
-- @include ./update.sql
-- @include ./delete.sql
-- @include ./list.sql
```

---

## Entity Conventions

Each operation gets its own file. Each file is self-contained:

```sql
CREATE SCHEMA IF NOT EXISTS users;

DROP FUNCTION IF EXISTS users.create(users.create_request);
DROP TYPE IF EXISTS users.create_request;
DROP TYPE IF EXISTS users.create_response;

CREATE TYPE users.create_request AS ( ... );
CREATE TYPE users.create_response AS ( ... );

CREATE OR REPLACE FUNCTION users.create(req users.create_request)
RETURNS users.create_response AS $$ ... $$ LANGUAGE plpgsql;
```

### Naming

- Schema: `users`
- Types: `users.create_request`, `users.create_response`
- Function: `users.create(req users.create_request)`
- List functions return `SETOF`: `RETURNS SETOF users.list_response`

### Drop Order

Functions must be dropped before their types. Never use `CASCADE`.

---

## Runner (Go)

### Schema migrations
1. Create a `schema_migrations` table if it doesn't exist
2. Read `migrations/schema/` in filename order
3. Skip files already recorded in `schema_migrations`
4. Execute and record each new file in a transaction

### Entity migrations
1. Read `migrations/entities/main.sql`
2. Parse `-- @include` directives recursively, resolving paths relative to the including file
3. Deduplicate — track visited files, skip if already loaded
4. Execute all resolved files in order against the database

### Runner warnings
- Warn about any `.sql` file under `migrations/entities/` not reachable from `main.sql`
