# Database Best Practices (pgx)

Postgres is accessed through `pgx/v5` + `pgxpool`. There is no ORM: every query
is explicit SQL in a repository, and golang-migrate owns the schema.

## 1. Querying

```go
// ✓ Always pass ctx; always use placeholders (never string-built values)
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

u, err := scanUser(r.pool.QueryRow(ctx, selectColumns+` FROM users u WHERE u.id=$1`, id))
```

- NULL → plain string fails to scan. Widen the column to a pointer in
  `scanUser`, then deref, or `coalesce(...)` in SQL.
- Identifiers (table/column in `ORDER BY`, dynamic SET lists) cannot be
  parameterized — whitelist them (`allowedSortColumns`).
- Translate driver errors at the repository boundary (`database.IsUniqueViolation`
  → `domain.ErrDuplicateEntry`, `pgx.ErrNoRows` → `domain.ErrNotFound`) so no
  caller imports pgx.

## 2. Transactions

```go
err := database.InTx(ctx, r.pool, func(tx pgx.Tx) error {
    if _, err := tx.Exec(ctx, `DELETE FROM user_roles WHERE user_id=$1`, id); err != nil {
        return err
    }
    _, err := tx.Exec(ctx, `INSERT INTO user_roles ...`)
    return err
})
```

`database.InTx` commits on success and rolls back on error or panic. Use it for
any multi-statement write (register + role assignment, role replacement).

## 3. Pool

`DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` / `DB_CONN_MAX_LIFETIME` configure the
pool in `internal/infra/database`. Defaults: 100 max, 30 min idle, and a
liveness `Ping` on open so an unreachable host fails at startup.

## 4. Schema

- SQL migrations only (`make migrate-create`).
  There is no AutoMigrate.
- Unique-ness on soft-deleted rows uses partial indexes
  (`WHERE deleted_at IS NULL`) so a deleted email can be reused.
- Role/permission changes go through `user_roles` / `role_permissions`; never
  denormalize a role list onto `users`.
