// Package remoteplayback turns a Part's stored location into somewhere the
// bytes can actually be fetched from.
//
// It is a consumer rather than a source: it acts on what materialising created
// instead of bringing content in. Without a consumer a library is inert — a
// user can add a film and nothing can play it (platform#24, platform#25).
//
// It is its own Go module (github.com/mosaic-media/module-remote-playback)
// importing only the published SDK (contracts/platform/v1) and the standard
// library, compiled into the Platform binary and resolved through the
// capability registry (platform#4, sdk#1).
//
// It resolves and never serves. The Platform owns transports (platform#3), so a
// module holds no listener: this returns a location, and the Platform's own
// origin fetches it, applies any headers the source needs and relays the bytes
// to the client. That is also what keeps a debrid credential server-side and the
// viewer's IP off the CDN.
//
// What it covers today is the direct path: a location that is already an http(s)
// URL. That is not a toy case — a Stremio aggregator such as AIOStreams resolves
// its own debrid backend (TorBox, Real-Debrid, …) and hands back a plain URL,
// which is the mainstream way remote content is watched. A bare magnet has no
// answer here and says so; resolving one needs a torrent engine, which is
// deliberately deferred rather than missing.
//
// It owns no schema (platform#8) and writes nothing. Playback position and
// watched state are Platform-owned (platform#26), because a second player, or an
// export module, has to read the same state.
package remoteplayback
