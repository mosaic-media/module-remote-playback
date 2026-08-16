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
	// modulePath is this module's import path. v1.ModuleVersion (SDK v0.12.0)
	// reads the version actually linked out of the build graph from it, so no
	// hand-maintained version constant has to be remembered at release time.
	modulePath = "github.com/mosaic-media/module-remote-playback"
)

// These assertions fail to compile if Capability drifts from the SDK contract
// the Platform resolves it through, or from the role it claims to fill (sdk#2).
var (
	_ v1.Capability       = (*Capability)(nil)
	_ v1.PlaybackProvider = (*Capability)(nil)
)

// Capability is the remote playback module, filling platform#25's RolePlayback.
// It is stateless: everything it needs arrives in the request, so it stays a
// pure function of the Part it is handed.
type Capability struct{}

// New builds the capability. It takes no arguments: module settings
// (platform#17) reach a capability per request rather than at construction, so
// the debrid and torrent paths will be configured that way too.
func New() *Capability { return &Capability{} }

// Manifest declares the module's identity and the single consumer role it
// fills.
func (c *Capability) Manifest() v1.Manifest {
	return v1.Manifest{
		ID:      CapabilityID,
		Version: v1.ModuleVersion(modulePath),
		Name:    "Remote Playback",
		Description: "Resolves a title into something playable and hands back where it lives. It " +
			"serves no bytes itself — the stream comes from wherever it already is.",
		Provides: []v1.Role{v1.RolePlayback},
	}
}

// Import is the base Capability's write verb, and a consumer acts on the
// materialised graph rather than adding to it. It refuses rather than returning
// an empty success: a caller routing an import here has made a mistake, and a
// silent no-op would hide it.
func (c *Capability) Import(_ context.Context, _ v1.ContentService, _ v1.ImportRequest) (v1.ImportResult, error) {
	return v1.ImportResult{}, fmt.Errorf("%s is a playback module and imports no content", CapabilityID)
}

// Resolve turns a Part's stored location into somewhere the bytes can be
// fetched from.
//
// Today that is a pass-through for a location that is already an http(s) URL.
// An aggregator addon pointed at a debrid backend returns exactly such a URL, so
// this covers the mainstream remote path without the module holding any debrid
// credential itself.
//
// The Part is a hint about what to play, not a durable address: a debrid URL
// snapshotted at import has very likely expired. That is why the Platform calls
// this at play time rather than reading the location out of the graph, so do not
// add caching here that would defeat it. Re-resolving through the source instead
// of trusting the snapshot is a deferred slice.
func (c *Capability) Resolve(_ context.Context, req v1.PlaybackRequest) (v1.PlaybackResolution, error) {
	loc := req.Part.Location

	if loc.Scheme == v1.LocalLocation {
		// A local file is a different consumer's job. Saying so is better than
		// resolving a filesystem path into a URL nothing can fetch.
		return v1.PlaybackResolution{}, fmt.Errorf("%s plays remote locations; this part is a local file", CapabilityID)
	}

	ref := strings.TrimSpace(loc.Ref)
	if ref == "" {
		return v1.PlaybackResolution{}, fmt.Errorf("part has no location reference to resolve")
	}

	if isMagnet(ref) {
		// A magnet needs either a torrent engine or a debrid service to become
		// bytes, and this module has neither yet. Naming the missing piece is
		// better than returning a URL nothing can open.
		return v1.PlaybackResolution{}, fmt.Errorf(
			"this release is a torrent, and no torrent engine or debrid service is configured to resolve it")
	}

	if !isHTTPURL(ref) {
		return v1.PlaybackResolution{}, fmt.Errorf("unsupported location reference %q", truncate(ref, 40))
	}

	return v1.PlaybackResolution{Kind: v1.PlaybackDirect, URL: ref}, nil
}

// isMagnet reports whether a reference is a magnet URI. It is checked before the
// general URL parse because a magnet is itself a valid URI and would otherwise
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

// truncate bounds a reference before it reaches an error message. A resolved
// location may be a signed URL carrying a credential and errors reach logs, so
// every failure path that quotes a ref must route it through here.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
