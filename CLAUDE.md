# Claude Instructions — module-remote-playback

This repository is a **core Mosaic module**: it is compiled into the Platform's
binary rather than installed into a running one. Its `release.yml` dispatches
`core-module-release`, and that dispatch is what moves the `require` line in the
Platform's `go.mod` — which is why there is no `cmd/` here and no binaries job,
because nothing serves this module out of process. Which tier clause it falls
under is
[architecture#3](https://github.com/mosaic-media/architecture/blob/main/docs/adr/0003-two-module-tiers.md)'s
business; the consequences for working here are below.

**The tier is a delivery and coupling decision, not a contract decision.** This
module is built exactly as a third party's would be — its own Go module,
importing only the published SDK, resolved through the Platform's capability
registry. It does not know which tier it is in, and moving it between tiers would
be a build change rather than a rewrite. So the boundary discipline is *stricter*
here, not looser: a dependency added here is one the Platform and every other
core module must resolve compatibly.

It is a **consumer**, not a source. A source brings content in; this one acts on
what materialising created, turning a `Part`'s stored location into somewhere the
bytes can actually be fetched from
([platform#25](https://github.com/mosaic-media/platform/blob/main/docs/adr/0025-playback-consumer-and-media-origin.md)).
`Manifest` declares `RolePlayback` and nothing else, and `Import` refuses rather
than returning an empty success, because a caller routing an import here has made
a mistake and a silent no-op would hide it.

## The boundary is the point

- **Import only [`sdk`](https://github.com/mosaic-media/sdk) and the standard
  library.** `boundary_test.go` parses every import and fails on anything else.
  Go already rejects a Platform-internal import because this is a separate
  module; the test keeps the intent explicit and catches a third-party dependency
  creeping in too. **It reads this directory only, not the tree** — which is
  sound while every source file is here, and stops being sound the moment a
  subdirectory is added. Adding one means making that test walk, or it declares
  the boundary clean while never reading the new file.
- **It matters more here than in a source module.** A consumer is the one that
  wants to reach the network, hold a credential and eventually touch the disk, so
  it is the one most likely to grow a dependency it should not have — and it is
  exactly the kind of module a third party will write.
- **This module resolves; it never serves bytes.** The Platform hosts the origin,
  mints the ticket and relays. Putting the byte path here would hand a module the
  thing the whole contract keeps away from it, and would put an ffmpeg dependency
  behind the SDK boundary. **If a change here starts to look like an HTTP
  handler, stop.**
- **It owns no schema and writes nothing.** Playback *state* is Platform-owned
  ([platform#26](https://github.com/mosaic-media/platform/blob/main/docs/adr/0026-playback-state-is-platform-owned.md)),
  and a module owning a store is ruled out
  ([platform#8](https://github.com/mosaic-media/platform/blob/main/docs/adr/0008-capabilities-do-not-own-stores.md)).
  A second player, or an export module, has to read the same state.
- **A facility a consumer wants is a Platform facility reached declaratively.**
  The things a consumer reaches for — a byte path, a fetcher, a credential store
  — are implementations, and the answer is never a primitive added to the SDK.
  What the SDK will and will not carry is
  [`sdk`](https://github.com/mosaic-media/sdk)'s own rule and its own records;
  read them there rather than trusting a summary written here.
- **MIT-licensed**, the author's choice, unlike the Platform's AGPL
  ([architecture#1](https://github.com/mosaic-media/architecture/blob/main/docs/adr/0001-licensing.md)).
  Files here carry no SPDX header — match the files already present.

## Fail with the reason named

A `Part` this module cannot resolve produces an **honest, specific failure** —
not an empty success, and not a generic error. The message is what a user sees:
a magnet says a torrent engine or debrid service is what it would need, a local
file says that is a different consumer's job, and an unsupported reference is
quoted so it can be recognised.

**Refs are truncated before they reach error text** (`truncate`), because errors
reach logs and a resolved location may be a signed URL carrying a credential.
Adding a new failure path means routing its ref through the same helper.

The `Part` is a **hint about what to play, not a durable address.** A debrid URL
snapshotted at import has very likely expired, which is why the Platform calls
`Resolve` at play time rather than reading the location out of the graph. Do not
add caching here that would defeat that.

## Everything runs in the container, nothing runs on the host

**Do not run `go build`, `go test`, `go vet` or `gofmt` directly on this
machine.** This repository's gate runs inside its test container:

```bash
docker compose -f docker-compose.test.yml run --rm test
```

That runs gofmt, `go build ./...`, `go vet ./...` and `go test ./...` against the
Go version pinned in `docker-compose.test.yml`, which must stay equal to the one
in `go.mod`. `.github/workflows/verify.yml` runs the same four steps — **keep the
two in step**, or a gate passes locally and fails on push. Append `bash` for a
shell in the same environment.

**What the container is really protecting is the boundary.** A host with a
populated module cache, a `go.work` left over from cross-repo work, or a stray
`replace` can satisfy an import a third party's machine could not — and
`boundary_test.go` still passes, because the import resolved. The container
resolves from the proxy exactly as a consumer does, so the test means what it
claims.

## Versioning and release

A change is a **minor** bump, tagged and pushed. Pushing the tag is the whole
publish: `release.yml` re-runs the full gate against the tagged commit, checks
the tag is a semver version and that `go.mod`'s module path matches the
repository, then **proves a consumer can resolve it** by doing `go get` from a
throwaway module through the public proxy — because the proxy and checksum
database are eventually consistent with a just-pushed tag, and without that step
a bad publish surfaces as a broken build in the platform repository with nothing
pointing back here.

Only then does `dispatch` fire, asking the Platform to bump its `require` and to
re-check the repository graph. **Both dispatch steps fail hard when their token
is unset.** They used to warn and exit 0, which meant an unset token reported
green while no bump was ever sent; the release is already complete by then, so a
red run un-publishes nothing and is the only way a broken bump chain becomes
visible.

**A `replace` must never land in a commit.** Use one for local cross-repo work
and remove it before committing.

The module reports the version that was **actually linked**, via
`v1.ModuleVersion` reading the build graph — not a hand-maintained constant,
which nothing forces to agree with anything. The constant this replaced had
already drifted once: it read `0.0.1` against a `v0.1.0` tag.

## Decision records

**This repository owns none, and has no `docs/adr/`. That is correct rather than
an oversight** — every decision that governs this module is stewarded where its
mechanism lives: the consumer role and the origin in
[`platform`](https://github.com/mosaic-media/platform), the contract surface in
[`sdk`](https://github.com/mosaic-media/sdk), the tier model in
[`architecture`](https://github.com/mosaic-media/architecture). Follow the links
rather than repeating what they say.

If a decision *does* become this repository's — one whose mechanism is a file
here — it earns `docs/adr/0001-…` and a generated `docs/adr/README.md` index.
The index script and the citation lint that `architecture` owns for the fleet are
**not vendored here**, so nothing in this repository's gate would check that the
index is current or that a citation resolves. Until they are, both are on you.

## Observability

Observability goes through the SDK's ambient `v1.Telemetry`
([sdk#5](https://github.com/mosaic-media/sdk/blob/main/docs/adr/0005-modules-observe-through-the-sdk.md)),
reached as `TelemetryFrom(ctx)`. **Do not print**, and do not configure an
exporter, a sink or retention — the Platform owns the observability plane. A
location reference is a credential: classify it, never write it verbatim.

<!-- shared-rules:begin -->
## Rules every Mosaic repository shares

*Generated. The source is `architecture/shared/repository-rules.md`; edit it there
and run `scripts/shared_rules.py --write` across the fleet. A copy edited in place
fails its repository's gate, which is the point: these rules were eleven
hand-kept copies in four variants, and the abridged ones had quietly dropped the
reasoning while keeping the rules — and in one case dropped a rule outright.*

### What this file may say

**A `CLAUDE.md` states rules, and facts about its own repository. It does not
state facts about another one — it links instead.**

An audit of all twelve of these files against their source found 74 stale claims.
None of roughly 180 rules was wrong; 62 of the 74 were facts about somebody
else's repository. Ownership predicts rot: a fact about this repository stays true
because whoever changes the code changes the sentence in the same session, and a
fact about another one dies the moment they edit it with nothing here going red.

The same applies to facts this repository already publishes in a generated
artefact — counts, versions, what is built. Point at the artefact.

### Decision records live with the code they govern

Each repository owns the records whose *mechanism* it holds — the spec file, the
lint gate, the conformance corpus, the composition root, the release workflow.
A decision can bind five repositories and still have exactly one steward.

- **`docs/adr/`**, numbered from 1 in every repository, with `docs/adr/README.md`
  a **generated** index. Read the index first; it is the bounded thing.
- **A record's heading carries no number.** The number lives in the filename and
  the index only, so a record's anchor survives being renumbered.
- **Cite a record as `repo#N`, and make it a link** — a relative path within a
  repository, an absolute URL across them, and the bare label only where no URL
  is possible, such as a code comment or a Dockerfile. The old `ADR NNNN`
  spelling is refused by a lint: once every repository numbers from 1, that form
  resolves quietly to a *different* record instead of dangling, and no tool in
  the fleet could detect it.
- **Cross-cutting records stay in [`architecture`](https://github.com/mosaic-media/architecture)** —
  the ones with no enforcing mechanism anywhere: licensing, repository naming and
  topology, the module tier model.

### Decision records are append-only

An ADR is an account of what was decided and why, at a time. It is evidence, not
documentation, and its value is that it was not edited afterwards.

- **Never rewrite a record's body** — not to correct it, not to annotate it, not
  to add "as built, this differs". That turns a record into a running commentary
  and destroys the thing it is for.
- **State changes go in the `**Status:**` line and nowhere else** — built, built
  in part (naming the part), or superseded, wholly or partly.
- **A changed decision earns a new record that supersedes it**, with its own
  Context / Decision / Alternatives / Consequences, and both records then point
  at each other through their Status lines. The old body stays exactly as it was.
- **An unbuilt decision is not a superseded one.** "Not done yet" belongs in the
  Status line and the roadmap; only a reversal earns a new record.

### The roadmap is maintained, not consulted

**`docs/roadmap.md` in [`architecture`](https://github.com/mosaic-media/architecture)
is the single record of where the build is, across every repository.** It stays
there because a milestone spans repositories by construction. Read it before
starting, and **update it in the same session as the change that dates it** — not
in a follow-up, which does not happen.

- A slice that lands is marked landed, **with what it left out named in the same
  sentence**. "Built" with no qualifier claims the whole slice shipped.
- Implementation that departed from its record is recorded where it departed.
  The surprises are the most valuable thing in it.
- **Do not restate the roadmap here.** A second copy of "what is built" in a
  `CLAUDE.md` is how the first copy goes stale unnoticed.
- A capability with no client path is not done — it is
  [owed](https://github.com/mosaic-media/architecture/blob/main/docs/unreachable-capability.md).

### Demonstrated, not asserted

**Say what you actually ran.** A skipped test is not a passed test, and "it should
work" is not evidence.

Each repository's container is the authority on its own gate, and the command is
in that repository's section below. It exists because the checks that matter fail
*soft*: a missing PostgreSQL skips storage tests and still prints `ok`, a missing
generator toolchain produces a drift guard that passes by not running. Where the
container cannot be run, running what you can on the host is better than running
nothing — **provided you report which checks ran and which did not.** Claiming a
gate passed when it was not executed is the one thing this rule exists to stop.

### Commit and push

- **Commit and push each repository separately.** They are siblings on disk and
  independent in git.
- **Commit author identity** must be `AdamNi-7080 <anicholls41@gmail.com>`. If git
  has no identity configured, set it repo-locally rather than globally.
- **Push once the change has been demonstrated working in this session.** Commit
  locally and say so otherwise. **Force-push always requires asking.**
<!-- shared-rules:end -->
