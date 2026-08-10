# module-remote-playback

Remote playback module for the [Mosaic](https://github.com/mosaic-media) platform — the first **consumer** module, built against the Mosaic SDK.

It is an **optional Mosaic module**: its own Go module that imports only
[`sdk`](https://github.com/mosaic-media/sdk) and the standard library, compiled
into a Mosaic Platform binary and resolved through the Platform's capability
registry. It owns no schema and writes nothing.

## What makes it different

Every module before it is a *source* — it brings content in and populates the
virtual plane. This one goes the other way: it acts on what materialising
created, turning a `Part`'s stored location into somewhere the bytes can
actually be fetched from. Until it existed the library was inert, and a user
could add a film that nothing could play
([platform#24](https://github.com/mosaic-media/platform/blob/main/docs/adr/0024-capability-gated-affordances.md),
[platform#25](https://github.com/mosaic-media/platform/blob/main/docs/adr/0025-playback-consumer-and-media-origin.md)).

## What it does

It fills `RolePlayback` with a single method. Given a `Part`, it returns a
location:

- **An `http(s)` location passes through** as a `Direct` resolution. That is the
  mainstream remote path rather than a special case — a Stremio aggregator such
  as AIOStreams resolves its own debrid backend (TorBox, Real-Debrid, …) and
  hands back a plain URL, so this module holds no debrid credential itself.
- **A magnet fails, with the reason.** Resolving one needs a torrent engine or a
  debrid service and it has neither, so it says so rather than returning a URL
  nothing can open. The message is what a user sees.
- **A local file is refused** — that is a different consumer's job.

**It resolves and never serves.** The Platform owns transports, so this returns
a location and the Platform's own origin fetches it, applies any headers the
source requires, and relays the bytes. That is also what keeps a debrid
credential server-side and the viewer's IP off the CDN.

The `Part` is a *hint about what to play*, not a durable address: a debrid link
snapshotted at import has very likely expired, which is why the Platform
resolves at play time rather than reading the location out of the graph.

## Build and test

**Everything runs in a container; nothing is built or tested on the host.**

```bash
docker compose -f docker-compose.test.yml run --rm test
```

That runs gofmt, `go build`, `go vet` and `go test` against a pinned toolchain,
resolving the published SDK from the module proxy exactly as a third party
would — which is what makes `boundary_test.go` mean something: this module
compiles against the SDK and the standard library and nothing else, and a host
with a stray `replace` or a leftover `go.work` could satisfy an import it should
not.

## Status

`v0.0.1`, the direct path only. Named and deferred, not missed:

- **The torrent engine** — the `Served` resolution, where the module produces
  bytes for the Platform to serve. It is what a magnet needs.
- **Candidate selection against a client capability profile**
  ([platform#27](https://github.com/mosaic-media/platform/blob/main/docs/adr/0027-stream-selection-against-a-client-profile.md)),
  which is what makes a browser able to play anything at all reliably.
- **Subtitle conversion** (SRT→VTT) through the same origin.

Playback position and watched state are Platform-owned and deliberately not
this module's
([platform#26](https://github.com/mosaic-media/platform/blob/main/docs/adr/0026-playback-state-is-platform-owned.md)):
a second player, or an export module, has to read the same state.

## License

MIT (see [`LICENSE`](LICENSE)) — a module may be licensed however its author
chooses; the Platform's AGPL-3.0 module-linking exception is what makes that
work.
