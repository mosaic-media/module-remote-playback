package remoteplayback

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	v1 "github.com/mosaic-media/sdk/contracts/platform/v1"
)

const (
	// CapabilityID is the id the Platform registers this module under.
	CapabilityID = "remote-playback"
	// modulePath is this module's import path, which is how it looks its own
	// version up in the build graph rather than carrying a constant that has to
	// be remembered at release time (SDK v0.12.0). The constant this replaces
	// had already drifted once — it read 0.0.1 against a v0.1.0 tag.
	modulePath = "github.com/mosaic-media/module-remote-playback"
)

// Capability satisfies the SDK's capability contract and the one role it
// declares. The assertions fail to compile if the module drifts from what the
// Platform resolves, or from a role it claims to fill (ADR 0027).
var (
	_ v1.Capability       = (*Capability)(nil)
	_ v1.PlaybackProvider = (*Capability)(nil)
)

// Capability is the remote playback module (ADR 0045's RolePlayback, first
// filled). It is stateless: everything it needs arrives in the request, which
// is what lets it stay a pure function of the Part it is handed.
type Capability struct{}

// New builds the capability. It takes no arguments today; when the debrid and
// torrent paths arrive they are configured through module settings (ADR 0021),
// which reach it per-request rather than at construction.
func New() *Capability { return &Capability{} }

// Manifest declares the module's identity and the single consumer role it
// fills.
func (c *Capability) Manifest() v1.Manifest {
	return v1.Manifest{
		ID:       CapabilityID,
		Version:  v1.ModuleVersion(modulePath),
		Name:     "Remote Playback",
		Provides: []v1.Role{v1.RolePlayback},
	}
}

// Import is the base Capability's write verb, and this module has nothing to do
// with it: a consumer acts on the materialised graph rather than adding to it.
// It refuses rather than returning an empty success, because a caller routing an
// import here has made a mistake and a silent no-op would hide it.
func (c *Capability) Import(_ context.Context, _ v1.ContentService, _ v1.ImportRequest) (v1.ImportResult, error) {
	return v1.ImportResult{}, fmt.Errorf("%s is a playback module and imports no content", CapabilityID)
}

// Resolve turns a Part's stored location into somewhere the bytes can be
// fetched from.
//
// Today that is a pass-through for a location that is already an http(s) URL,
// which is less trivial than it looks: an aggregator addon pointed at a debrid
// backend returns exactly such a URL, so this covers the mainstream remote path
// without the module holding any debrid credential itself.
//
// The Part is treated as a *hint about what to play*, not as a durable address.
// A debrid URL snapshotted at import has very likely expired by now, which is
// why the Platform calls this at play time rather than reading the location out
// of the graph — and why re-resolving through the source, rather than trusting
// the snapshot, is the next slice.
func (c *Capability) Resolve(_ context.Context, req v1.PlaybackRequest) (v1.PlaybackResolution, error) {
	loc := req.Part.Location

	if loc.Scheme == v1.LocalLocation {
		// A local file is a different consumer's job. Saying so is better than
		// resolving a filesystem path into a URL that cannot be fetched.
		return v1.PlaybackResolution{}, fmt.Errorf("%s plays remote locations; this part is a local file", CapabilityID)
	}

	ref := strings.TrimSpace(loc.Ref)
	if ref == "" {
		return v1.PlaybackResolution{}, fmt.Errorf("part has no location reference to resolve")
	}

	if isMagnet(ref) {
		// The honest answer, and the one the deferred torrent-engine slice
		// exists to replace. A source that hands back a magnet needs either a
		// torrent engine or a debrid service to turn it into bytes, and this
		// module has neither yet — so it fails with the reason rather than
		// returning a URL nothing can open.
		return v1.PlaybackResolution{}, fmt.Errorf(
			"this release is a torrent, and no torrent engine or debrid service is configured to resolve it")
	}

	if !isHTTPURL(ref) {
		return v1.PlaybackResolution{}, fmt.Errorf("unsupported location reference %q", truncate(ref, 40))
	}

	return v1.PlaybackResolution{Kind: v1.PlaybackDirect, URL: ref}, nil
}

// isMagnet reports whether a reference is a magnet URI. It is checked before
// the general URL parse because a magnet *is* a valid URI and would otherwise
// fall through to a less useful error.
func isMagnet(ref string) bool {
	return strings.HasPrefix(strings.ToLower(ref), "magnet:")
}

// isHTTPURL reports whether a reference is an absolute http(s) URL with a host.
// The host check matters: "http://" alone parses cleanly and would otherwise be
// handed to the origin as something to fetch.
func isHTTPURL(ref string) bool {
	u, err := url.Parse(ref)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// truncate bounds a reference before it reaches an error message, so a long
// signed URL does not end up quoted in full in a log.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
