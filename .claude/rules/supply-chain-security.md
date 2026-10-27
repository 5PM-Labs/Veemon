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
| Go (root module) | `go.sum` | `govulncheck ./...` (CI `vuln` job) |
| Bun (`apps/web`, `apps/ai`, `packages/*`) | `bun.lock` (`--frozen-lockfile` in CI) | `bun audit` / OSV-Scanner |
| Dart/Flutter (`apps/mobile`) | `pubspec.lock` | OSV-Scanner on `pubspec.lock` |

`bun audit` is a CI gate (`vuln-js` job), mirroring `vuln` for Go. Advisory
IDs listed under "Accepted risk" below are the only ones `bun audit --ignore`
may suppress; everything else fails the build.

### Runtime versions

- The JS runtime is **Bun**, pinned as `BUN_VERSION` in `.github/workflows/ci.yml`
  and mirrored by `.tool-versions`. Never `latest` in CI.
- Node upshots we do not run directly still matter: Bun vendors Node-compatible
  code (undici, c-ares, TLS, brotli) and `bun audit` reports Node CVEs against it.
  Treat a Node security release as a trigger to bump `BUN_VERSION` to a Bun
  build whose vendored Node is patched — not to add a Node install step.

### Accepted risk (unpatched advisories)

Every entry needs: advisory ID, why no upgrade path, proof the vulnerable code
is unreachable, owner, and re-check trigger. An entry without reachability
proof is not accepted — it is a bug to fix.

| Advisory | Path | Severity | Status |
|----------|------|----------|--------|
| GHSA-vfj7-8cjw-p6xm — `braces <=3.0.3`, stack exhaustion via deeply nested patterns | `workspace:@veemon/ai › awilix › fast-glob › micromatch › braces` | high | Accepted |

- **No upgrade path.** `braces@3.0.3` is the latest release and still
  vulnerable; upstream is unpatched. `awilix` only depends on `fast-glob`, so
  this is a transitive 4-deep pin, not an `awilix` choice.
- **Unreachable.** The DoS needs a caller passing an attacker-controlled glob
  pattern. `braces` is reached through `awilix`'s `loadModules`, which
  `apps/ai` never calls — `apps/ai/src/composition.ts:1` imports only
  `createContainer`, `asValue`, and `asFunction`, so the container is wired
  statically and no glob string ever exists. Verified:
  `grep -rn "loadModules\|GlobWithOptions\|globPatterns" apps/ai/src` → no hits.
- **Re-check trigger.** Remove this entry the moment either becomes true:
  `loadModules` is used anywhere in `apps/ai`, or `braces` ships a fix
  (then upgrade and drop the `--ignore` from the `vuln-js` job).

#### Why it is not tree-shaken away

The obvious fix is to drop `fast-glob` from the install. It fails, and the
next person will try it again — so record the result:

- `awilix@13.0.5` declares `fast-glob` as a hard `dependencies` (not a peer), and
  `lib/awilix.js` → `lib/list-modules.js` → `require("fast-glob")` is a module-load
  edge, not a lazy call inside `loadModules`. Bun resolves the whole static graph,
  so `braces` is installed (and audited) even though `loadModules` is never called.
- Rewriting the lock entry to `"optionalPeers": ["fast-glob"]` (to simulate the
  fix) does not resolve: Bun 1.4.3 strips the override on the next resolve and
  leaves a dangling symlink in the store — `error: ENOENT reading …/awilix@13.0.5/node_modules/fast-glob`,
  i.e. `import "awilix"` stops resolving at runtime. Reverted.
- `awilix/browser` has no `fast-glob`, but it is only selected via the
  `browser`/`workerd`/`react-native` export conditions; mapping `browser` in
  `apps/ai` would also swap the API for the browser build of `awilix` in the web
  bundle. Not worth the coupling for an unreachable DoS.

Fix condition: an `awilix` release that moves `fast-glob` to a peer/optional
dependency, or `braces > 3.0.3`. Bump, then delete this entry and the
`--ignore` in `ci.yml`.

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
