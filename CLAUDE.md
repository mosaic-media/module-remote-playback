# Claude Instructions — module-remote-playback

This repository is an **optional Mosaic module**, and the first **consumer** one
([ADR 0045](https://github.com/mosaic-media/architecture/blob/main/docs/adr/0045-playback-consumer-and-media-origin.md)).
Every module before it is a *source* that brings content in; this one acts on
what materialising created, turning a `Part`'s stored location into somewhere the
bytes can actually be fetched from.

It is built exactly as a third party's module would be — its own Go module,
importing only the SDK — because building it the third-party way is what proves
the third-party way exists.

## The boundary is the point

- **Import only [`sdk`](https://github.com/mosaic-media/sdk) and the standard
  library.** `boundary_test.go` parses every import and fails on anything else.
  Go already rejects a Platform-internal import because this is a separate
  module; the test keeps the intent explicit and catches a third-party dependency
  creeping in.
- **It matters more here than in a source module.** A consumer is the one that
  wants to reach the network, hold a credential and eventually touch the disk, so
  it is the one most likely to grow a dependency it should not have.
- **This module resolves; it never serves bytes.** The Platform hosts the origin,
  mints the ticket and relays (ADR 0045). Putting the byte path here would hand a
  module the thing the whole contract keeps away from it, and would put an ffmpeg
  dependency behind the SDK boundary. If a change here starts to look like an
  HTTP handler, stop.
- **The same line runs through the SDK itself: it names no implementation and
  depends on nothing**
  ([ADR 0135](https://github.com/mosaic-media/architecture/blob/main/docs/adr/0135-the-sdk-carries-no-implementation.md)).
  The SDK says how a module interacts with the Platform; the Platform holds the
  implementations. A consumer feels this first, because the things a consumer
  wants — a byte path, a fetcher, a credential store — are implementations, and
  the answer is a Platform facility reached declaratively, not a primitive added
  to the contract.
- **It owns no schema and writes nothing.** Playback *state* is Platform-owned
  ([ADR 0046](https://github.com/mosaic-media/architecture/blob/main/docs/adr/0046-playback-state-is-platform-owned.md)),
  and a module owning a store is ruled out
  ([ADR 0012](https://github.com/mosaic-media/architecture/blob/main/docs/adr/0012-capabilities-do-not-own-stores.md)).
- **Fail with the reason named.** A magnet this module cannot resolve is an
  honest, specific failure — not an empty success, and not a generic error. Refs
  are truncated in error text because they reach logs and may carry a credential.
- **MIT-licensed**, the author's choice, unlike the Platform's AGPL
  ([ADR 0022](https://github.com/mosaic-media/architecture/blob/main/docs/adr/0022-licensing.md)).

## Versioning and release

The Platform requires this at a **tagged version with no `replace`** — a
`replace` must never land in a commit. A change is a minor bump, tagged and
pushed, then the Platform's `go.mod` require is bumped to match.

```bash
git tag v0.3.0 && git push origin main && git push origin v0.3.0
```

The module reports the version that was **actually linked**, via
`v1.ModuleVersion` reading the build graph — not a hand-maintained constant,
which nothing forces to agree with anything.

## Everything runs in the container, nothing runs on the host

**Do not run `go build`, `go test`, `go vet` or `gofmt` directly on this
machine.** This repository's gates run inside its test container:

```bash
docker compose -f docker-compose.test.yml run --rm test
```

That runs gofmt, `go build ./...`, `go vet ./...` and `go test ./...` against
the Go version pinned in the compose file, which must stay equal to the one in
`go.mod`. Append `bash` for a shell in the same environment.

**What the container is really protecting is the boundary.** This module
compiles against the published SDK and the standard library and nothing else,
and `boundary_test.go` enforces that by parsing every import. A host with a
populated module cache, a `go.work` left over from cross-repo work, or a stray
`replace` can satisfy an import that a third party's machine could not — and the
boundary test still passes, because the import resolved. The container resolves
from the proxy exactly as a consumer does, so the test means what it claims.

## Workflow

- Commit and push this repository **separately** from `platform`.
- **Commit author identity** must be `AdamNi-7080 <anicholls41@gmail.com>`.
- The test container green before pushing.
- Observability goes through the SDK's ambient `v1.Telemetry`
  ([ADR 0059](https://github.com/mosaic-media/architecture/blob/main/docs/adr/0059-modules-observe-through-the-sdk.md)),
  reached as `TelemetryFrom(ctx)`. Do not print, and do not configure an
  exporter, a sink or retention — the Platform owns the observability plane.

## The roadmap and the decision records

These rules are identical in every Mosaic repository. They exist because the
state of the build and the reasons behind it are the two things that rot fastest
and report nothing when they do — no build fails, no test goes red.

### The roadmap is maintained, not consulted

**`docs/roadmap.md` in [`architecture`](https://github.com/mosaic-media/architecture)
is the single record of where the build is.** Read it before starting work, and
**update it in the same session as the change that dates it** — not in a
follow-up, which does not happen.

- **A slice that lands is marked landed, with what was left out.** "Built" with
  no qualifier is a claim that the whole slice shipped; if part of it did not,
  say which part and why in the same sentence.
- **Implementation that departs from the plan is recorded where it departed.**
  The roadmap is derived from the code, not from the intention that preceded it,
  and the surprises are the most valuable thing in it.
- **Do not restate the roadmap here.** A second copy of "what is built" in a
  `CLAUDE.md` is how the first copy goes stale unnoticed. This file carries how
  to work in *this* repository; the roadmap carries what has been done across all
  of them.
- **A capability with no client path is not done — it is
  [owed](https://github.com/mosaic-media/architecture/blob/main/docs/unreachable-capability.md).**
  If you delete or fail to build a client path to a working service, add its row
  to that register in the same change.

### Decision records are append-only

An ADR is an account of what was decided and why, at a time. It is evidence, not
documentation, and its value is that it was not edited afterwards.

- **Never rewrite a record's body to match what was built.** Not to correct it,
  not to annotate it, not to add "as built, this differs". That pattern turns a
  record into a running commentary and destroys the thing it is for.
- **State changes in the `**Status:**` line, and nowhere else.** That is where a
  record says it is built, built in part (naming the part), or superseded —
  wholly ("Superseded by ADR N") or partly ("Partly superseded: X was reversed by
  ADR N; the rest stands").
- **A changed decision needs a new record that supersedes it.** If the code
  deliberately does something a record decided against, that is a decision and it
  is written down as one, with its own Context / Decision / Alternatives /
  Consequences. Both records then stand: the old one keeps its reasoning, the new
  one carries the change.
- **An unbuilt decision is not a superseded one.** "We have not done this yet"
  belongs in the Status line and the roadmap. Only a genuine reversal earns a new
  record.
- **Records live only in `architecture/docs/adr/`**, numbered sequentially in
  kebab-case. Adding one means adding it to `nav:` in `mkdocs.yml`, and
  `mkdocs build --strict` must pass.

**If the code and a record disagree, say so rather than quietly picking one.** An
honest "this is unresolved" is worth more than a plausible reconciliation that
reads as settled.
