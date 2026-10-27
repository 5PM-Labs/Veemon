# Common Commands

Run these `make` targets from the repo root (or via the root `bun run *:api` scripts).

```bash
# Run
make run              # API server (HTTP + gRPC)
make run-worker       # RabbitMQ worker
make dev              # server with hot reload (Air)

# Build
make build            # server binary
make build-worker     # worker binary

# Quality (match CI)
make lint             # golangci-lint (v2)
make test             # go test -race
make test-coverage    # + coverage profile

# Docker compose (runs the migrate step, then the app)
make compose-up
make compose-down

# Generate proto (regenerate Go + TS from contract/*.proto)
make proto

# Database migrations (golang-migrate is the source of truth)
make migrate                                # apply pending migrations
make migrate-create name=create_orders_table
make seed
make fresh-seed                             # drop, migrate, seed
```

## Mobile (`apps/mobile`, Flutter)

```bash
make -C apps/mobile gen        # pub get + gen-l10n + build_runner
make -C apps/mobile check      # format-check + analyze (0 issues) + test — matches CI
make -C apps/mobile run API_BASE_URL=http://10.0.2.2:3000
make -C apps/mobile build-apk  # compile check
```

Monorepo-wide Bun scripts (repo root): `bun run proto`, `bun run dev`,
`bun run build`, `bun run test`. See [commands in the README](../../README.md#make-commands).
