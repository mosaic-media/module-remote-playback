# Claude Instructions — module-remote-playback

A Mosaic module filling `RolePlayback`: given a `Part`, it returns where the
bytes can be fetched from. It is a **consumer** — it acts on the materialised
graph rather than bringing content in, and it serves no bytes itself.
`README.md` describes what it resolves today.

Fleet-wide conventions — commits, decision records, citation form, the roadmap —
are in [`architecture`](https://github.com/mosaic-media/architecture/blob/main/CLAUDE.md).
This file is what is specific to `module-remote-playback`.

## It is a core module, compiled into the Platform binary

The evidence visible from here is `release.yml`, whose `dispatch` job sends
`core-module-release`, and the absence of a `cmd/`: nothing serves this module
out of process. The comment on that job still hedges between the two tiers —
the question is settled, and the comment is what is out of date.

**The tier is a delivery and coupling decision, not a contract one.** This module
is written exactly as a third party's would be, does not know which tier it is
in, and would move tiers as a build change rather than a rewrite. So the boundary
below is *stricter* here than in an optional module, not looser.

## The boundary

- **Only the published [`sdk`](https://github.com/mosaic-media/sdk) and the
  standard library.** Read `go.mod` for what is required and at what version;
  there is no `contracts` dependency and no third-party one. It shares the
  Platform binary's dependency graph, so anything added here is something the
  Platform and every other core module must resolve compatibly. A consumer is the
  module most likely to want one it should not have: it is the kind that reaches
  the network, holds a credential and eventually touches the disk.
- `boundary_test.go` makes that executable by parsing every import — but it
  **reads this directory only, and skips `_test.go` files.** It does not walk
  the tree. **Adding a subdirectory means making it walk in the same change**,
  or the boundary is reported clean while the new file is never read. It also
  fails when it finds no non-test file at all, so it cannot pass by checking
  nothing.
- Go already rejects a Platform-internal import, this being a separate module;
  the test keeps the intent explicit and catches a third-party import too.

## What the capability declares

`Manifest()` provides exactly `[v1.RolePlayback]`, and
`TestManifestDeclaresOnlyThePlaybackRole` pins it. Compile-time assertions in
`capability.go` tie `Capability` to the SDK's `v1.Capability` and
`v1.PlaybackProvider`, so drift from either is a build failure rather than a
registration that fails at runtime.

The manifest's version is `v1.ModuleVersion(modulePath)` — the version actually
linked, read from the build graph. **Do not add a version constant**; nothing
would force it to agree with the tag.

`New()` takes no arguments and the capability is stateless: everything a resolve
needs arrives in the request, and module settings reach a capability per
invocation rather than at construction.

## `Import` refuses, and that is deliberate

A consumer materialises nothing, so `Import` returns an error rather than an
empty success — a caller routing an import here has made a mistake, and a silent
no-op would hide it. `TestImportRefuses` pins it.

## Failures name the reason, and never quote a whole reference

Each unresolvable case fails with the missing piece named: a magnet says a
torrent engine or debrid service would be needed, a local file says that is a
different consumer's job, an unsupported reference is quoted back. The magnet
check runs before the URL parse, because a magnet is itself a valid URI and
would otherwise fall through to a less useful error.

**Every failure path that quotes a reference must route it through `truncate`.**
A resolved location can be a signed URL carrying a credential, and errors reach
logs a user pastes into an issue. `TestErrorsDoNotQuoteAWholeCredentialedURL`
pins the truncation; a new quoting path needs the helper and a case beside it.

## What must not grow here

- **It resolves; it never serves.** The Platform owns transports and its own
  origin fetches and relays the bytes. If a change here starts to look like an
  HTTP handler, stop.
- **It owns no schema and writes nothing.** Playback state is Platform-owned.
- **No caching of a resolved location.** The `Part` is a hint about what to
  play, not a durable address — a debrid URL snapshotted at import has very
  likely expired, which is why the Platform resolves at play time.
- **The facilities a consumer reaches for are Platform facilities, reached
  declaratively.** A byte path, a fetcher, a credential store are
  implementations, and the answer is never a primitive added to
  [`sdk`](https://github.com/mosaic-media/sdk) — whose own rule says so; read it
  there.

## The gate

```bash
docker compose -f docker-compose.test.yml run --rm test
```

That runs the citation lint, gofmt, `go build`, `go vet` and `go test` on the Go
image pinned in `docker-compose.test.yml`, which must stay equal to `go.mod`'s
`go` directive — bump both together. The command installs a Python first when
the image has none, rather than letting the lint silently not run. Append `bash`
for a shell in the same environment.

`.github/workflows/verify.yml` is what refuses a push, and it runs the same
checks with `setup-go` on a fresh runner rather than nesting a container in one.
**Keep the two in step.** `release.yml` reuses that workflow through
`workflow_call`, so a tag cannot publish something unverified.

What the container protects is the boundary: a host with a populated module
cache, a stray `replace` or a leftover `go.work` can satisfy an import a third
party's machine could not, and `boundary_test.go` still passes because the
import resolved.

## Release

A pushed `v*` tag is the whole publish. `release.yml` reuses the gate, checks
the tag is `vMAJOR.MINOR.PATCH`, checks `go list -m` matches the repository,
creates the GitHub release, then **proves a consumer can resolve the version**
by doing `go get` from a throwaway module through the public proxy — retried,
because the proxy and checksum database are eventually consistent with a
just-pushed tag.

Only then does `dispatch` fire, sending `core-module-release` and `graph-check`
to [`platform`](https://github.com/mosaic-media/platform). **Both steps fail
hard when `PLATFORM_DISPATCH_TOKEN` is unset** rather than reporting green while
nothing was sent; the release is already complete by then, so a red run
un-publishes nothing.

**A release reaches a Platform through that `go.mod` bump**, not through the
signed index in [`registry`](https://github.com/mosaic-media/registry): there are
no binaries to catalogue. A move out of process would change the shape of that
job.

For a local cross-repo loop use a `replace` in the *consumer's* `go.mod` and
remove it before committing; **a `replace` must never land in a commit.**

## Conventions

- **MIT**, and files here carry **no SPDX header** — match the files already
  present rather than importing the Platform's convention.
- Observability goes through the SDK's ambient `v1.Telemetry` (`TelemetryFrom`).
  Do not print, and configure no exporter or sink. A location reference is a
  credential there too: classify it, never write it verbatim.
- `scripts/adr_lint.py` is vendored and says so in its header. Do not edit it
  here; change it at its source and re-vendor.
- **This repository owns no decision records and has no `docs/adr/`.** Every
  decision governing this module is stewarded where its mechanism lives — cite
  it as a link rather than restating what it says. `pages.yml` publishes this
  repository to GitHub Pages on a push to `main` through a reusable workflow
  held in `architecture`.
