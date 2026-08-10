// Package remoteplayback is the remote playback module: the first *consumer*
// Mosaic module. It is its own Go module
// (github.com/mosaic-media/module-remote-playback) importing only the published
// SDK (contracts/platform/v1) and the standard library, compiled into the
// Platform binary and resolved through the capability registry (platform#4,
// sdk#1, platform#25).
//
// Every module before it is a *source*: it brings content in and populates the
// virtual plane. This one does the opposite — it acts on what materialising
// created, turning a Part's stored location into somewhere the bytes can
// actually be fetched from. Until it existed the library was inert: a user
// could add a film and nothing could play it (platform#24).
//
// It resolves and never serves. The Platform owns transports (platform#3), so a
// module has no business holding a listener; this returns a location and the
// Platform's own origin fetches it, applies any headers the source needs, and
// relays the bytes to the client — which is also what keeps a debrid
// credential server-side and the viewer's IP off the CDN.
//
// What it covers today is the direct path: a location that is already an
// http(s) URL. That is not a toy case — a Stremio aggregator such as AIOStreams
// resolves its own debrid backend (TorBox, Real-Debrid, …) and hands back a
// plain URL, which is the mainstream way people actually watch remote content.
// A bare magnet has no answer here and says so plainly; resolving one needs a
// torrent engine, which is a named, deferred slice, not an oversight.
//
// It owns no schema (platform#8) and writes nothing. Playback position and
// watched state are Platform-owned and deliberately not this module's
// (platform#26): a second player, or an export module, must read the same state,
// and state that dies with the module would be the wrong shape.
package remoteplayback
