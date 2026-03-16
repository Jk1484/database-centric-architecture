# sqlm Roadmap

**sqlm (Modular SQL)** is a compiled language that brings a Go-like module system, type checking, and tooling to PostgreSQL. Source files use the `.sqlm` extension and compile to a single `.sql` file. Execution against PostgreSQL is a CI/CD concern — not part of the toolchain.

## CLI

```
sqlm build    -- compile .sqlm → single .sql file
sqlm lint     -- static analysis
sqlm lsp      -- start the language server
```

## Language

| sqlm | Go equivalent |
|---|---|
| `package users` | `package users` |
| `import "queries"` | `import "queries"` |
| `func init() {}` | `func init() {}` |
| `main.sqlm` | `main.go` entry point |
| `package users` (also) | `CREATE SCHEMA IF NOT EXISTS users` |
| `.sqlm` | `.go` |
| compiled `.sql` | binary |

## File Structure

```
migrations/entities/
  main.sqlm                        -- entry point, imports packages
  users/
    create.sqlm                    -- package users
    get.sqlm
    update.sqlm
    delete.sqlm
    list.sqlm
  queries/
    build_where_clause.sqlm        -- package queries
```

## Compiled Output Order Per Package

1. `CREATE SCHEMA IF NOT EXISTS <package>` — from `package` declaration
2. `func init()` bodies — in file order
3. raw SQL body — in file order

Imported packages are output before the importing package, recursively.

---

## Phase 1 — Parser ✅
> Foundation for everything else. Compiler, linter, and LSP all consume this.

- [x] Parse `package <name>` declaration
- [x] Parse `import "<package>"` directives
- [x] Parse `func init() {}` blocks
- [x] Resolve imports to directories by package name
- [x] Build a dependency graph (package → packages it imports)
- [x] Detect circular imports
- [x] Traverse from `main.sqlm` and produce an ordered file list

---

## Phase 2 — Compiler ✅
> Compiles `.sqlm` source into a single `.sql` file.

- [x] Auto-generate `CREATE SCHEMA IF NOT EXISTS <package>` from `package` declaration
- [x] Hoist all `func init()` blocks before body SQL
- [x] Concatenate packages in dependency order
- [x] Write final `.sql` output file

---

## Phase 3 — Linter
> Static analysis. Built on the parser.

- [ ] Warn about `.sqlm` files not reachable from `main.sqlm`
- [ ] Validate every file has a `package` declaration
- [ ] Validate that imported packages exist

---

## Phase 4 — LSP
> IDE integration. Built on the linter's index.

- [ ] Go to definition (click a type → jump to where it's defined)
- [ ] Find all references (see every function that uses a type)
- [ ] Unused file warning (file exists but not reachable from `main.sqlm`)
- [ ] Hover to show type shape
