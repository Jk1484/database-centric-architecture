# sqlm Roadmap

**sqlm (Modular SQL)** is a compiled language that brings Go-like module system, type checking, and tooling to PostgreSQL. Source files use the `.sqlm` extension and compile to a single `.sql` file executed against PostgreSQL.

## CLI

```
sqlm build    -- compile .sqlm → .sql and execute against PostgreSQL
sqlm lint     -- static analysis
sqlm lsp      -- start the language server
```

## Language

| sqlm | Go equivalent |
|---|---|
| `package users` | `package users` |
| `import "queries"` | `import "queries"` |
| `func init() {}` | `func init() {}` |
| `func create() {}` | `func create() {}` |
| `main.sqlm` | `main.go` entry point |
| `package users` (also) | `CREATE SCHEMA IF NOT EXISTS users` |
| `.sqlm` | `.go` |
| compiled `.sql` | binary |

## File Structure

```
migrations/entities/
  main.sqlm             -- entry point, imports packages
  users/
    create.sqlm         -- package users
    get.sqlm            -- package users
    update.sqlm         -- package users
    delete.sqlm         -- package users
    list.sqlm           -- package users
  queries/
    build_where_clause.sqlm  -- package queries
```

## Execution Order Per Package

1. `CREATE SCHEMA IF NOT EXISTS <package>` — auto-generated from `package` declaration
2. `func init()` blocks — in file order
3. remaining `func` bodies — in file order

## Example File

```sqlm
package users

import "queries"

func init() {
    DROP FUNCTION IF EXISTS users.create(users.create_request);
    DROP TYPE IF EXISTS users.create_request;
    DROP TYPE IF EXISTS users.create_response;
}

func create() {
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
    RETURNS users.create_response AS $$
    DECLARE
        result users.create_response;
    BEGIN
        INSERT INTO public.users (name, email)
        VALUES (req.name, req.email)
        RETURNING id, name, email, created_at, updated_at
        INTO result.id, result.name, result.email, result.created_at, result.updated_at;

        RETURN result;
    END;
    $$ LANGUAGE plpgsql;
}
```

---

## Phase 1 — Parser
> Foundation for everything else. Runner, linter, and LSP all consume this.

- [ ] Parse `package <name>` declaration
- [ ] Parse `import "<package>"` directives
- [ ] Parse `func init() {}` blocks
- [ ] Parse `func <name>() {}` blocks
- [ ] Resolve imports to directories by package name
- [ ] Build a dependency graph (package → packages it imports)
- [ ] Detect circular imports
- [ ] Traverse from `main.sqlm` and produce an ordered file list

---

## Phase 2 — Compiler
> Compiles `.sqlm` source into a single executable `.sql` file.

- [ ] Auto-generate `CREATE SCHEMA IF NOT EXISTS <package>` from `package` declaration
- [ ] Hoist all `func init()` blocks after schema creation
- [ ] Concatenate `func` bodies in file order
- [ ] Concatenate packages in dependency order
- [ ] Write final `.sql` output file

---

## Phase 3 — Runner
> Executes compiled output against PostgreSQL.

### Schema migrations
- [ ] Create `schema_migrations` tracking table on first run
- [ ] Read `migrations/schema/` in filename order
- [ ] Skip already-applied files
- [ ] Execute and record each new file in a transaction

### Entity migrations
- [ ] Compile `migrations/entities/main.sqlm`
- [ ] Execute compiled output on every startup

---

## Phase 4 — Linter
> Static analysis. Built on the parser.

- [ ] Warn about `.sqlm` files not reachable from `main.sqlm`
- [ ] Validate that types referenced in functions are defined
- [ ] Validate drop order inside `func init()` (functions before types)
- [ ] Validate `SETOF` used for list function return types
- [ ] Validate every file has a `package` declaration

---

## Phase 5 — LSP
> IDE integration. Built on the linter's index.

- [ ] Go to definition (click a type → jump to where it's defined)
- [ ] Find all references (see every function that uses a type)
- [ ] Unused file warning (file exists but not reachable from `main.sqlm`)
- [ ] Hover to show type shape
