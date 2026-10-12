# Supply Chain & Secrets Security

Covers what we pull in (dependencies, actions, base images) and what we must
never leak (secrets). Maps to SOC 2 CC8/CC9 and OWASP A06/A08.

## Secrets

- **Never commit secrets** — not in code, tests, fixtures, `.env.example`,
  Dockerfiles, Flutter `--dart-define` defaults, or `apps/web` Vite env
  (anything `VITE_*` is shipped to the browser — it is public).
- Mobile/web clients hold **no server secrets** at all. A key embedded in an app
  binary is public; anything that must stay secret goes behind the Go API.
- Load secrets from the environment / secret manager (Infisical); validate at
  startup (`Config.Validate()` fails fast on weak or placeholder values).
- Enable GitHub secret scanning + push protection on the repo. If a secret is
  committed: **rotate first**, then purge history — rewriting history alone
  does not un-leak it.

## Dependencies

| Ecosystem | Lockfile (commit it) | Vulnerability scan |
|-----------|---------------------|-------------------|
| Go (`apps/api`, `packages/go-common`) | `go.sum` | `govulncheck ./...` (CI `vuln` job) |
| Bun (`apps/web`, `apps/ai`, `packages/*`) | `bun.lock` (`--frozen-lockfile` in CI) | `bun audit` / OSV-Scanner |
| Dart/Flutter (`apps/mobile`) | `pubspec.lock` | OSV-Scanner on `pubspec.lock` |

- Add a dependency only when it pulls its weight; prefer well-maintained,
  widely used packages (check publisher, last release, open advisories,
  license). Note the reason in the PR.
- No `replace`/`dependency_overrides`/git-URL dependencies pointing at forks
  without a comment explaining why and an owner.
- Licenses: no AGPL/GPL in shipped binaries without legal sign-off.
- Renovate/Dependabot keeps versions current; security updates merge within
  the SLA (critical ≤ 7 days, high ≤ 30 days).

## CI/CD hardening

- Pin third-party GitHub Actions to a full commit SHA (or at least a major tag
  from a verified publisher); give workflows `permissions:` with least privilege
  (`contents: read` by default).
- Don't run untrusted PR code with secrets (`pull_request_target` + checkout of
  the PR head is forbidden).
- Never `continue-on-error` or skip a lint/test/vuln job to merge.
- Build artifacts are produced by CI only, from a tagged commit; container
  images use pinned base-image digests and run as non-root.
- Generate an SBOM (e.g. `syft`) for release images and mobile builds; keep it
  with the release.

## Mobile release signing

- Android keystore and iOS signing certificates/provisioning profiles live in
  the CI secret store, never in the repo. `key.properties`, `*.jks`,
  `*.keystore`, `*.p12`, `*.mobileprovision` are git-ignored.
- Release builds are obfuscated (`--obfuscate --split-debug-info=...`); upload
  the debug symbols to the crash reporter, don't ship them.
