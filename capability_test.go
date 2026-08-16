package remoteplayback_test

import (
	"context"
	"strings"
	"testing"

	remoteplayback "github.com/mosaic-media/module-remote-playback"
	v1 "github.com/mosaic-media/sdk/contracts/platform/v1"
)

func remotePart(ref string) v1.Part {
	return v1.Part{
		Role:     v1.PartEdition,
		Location: v1.MediaLocation{Scheme: v1.RemoteLocation, Provider: "stremio", Ref: ref},
	}
}

func TestManifestDeclaresOnlyThePlaybackRole(t *testing.T) {
	m := remoteplayback.New().Manifest()

	if m.ID != remoteplayback.CapabilityID {
		t.Errorf("ID = %q, want %q", m.ID, remoteplayback.CapabilityID)
	}
	if len(m.Provides) != 1 || m.Provides[0] != v1.RolePlayback {
		t.Fatalf("Provides = %v, want exactly [%q]", m.Provides, v1.RolePlayback)
	}
}

// TestResolveDirectURL pins the direct path: an aggregator addon pointed at a
// debrid backend hands back a plain https URL, and the module passes it through
// for the Platform's origin to relay.
func TestResolveDirectURL(t *testing.T) {
	const ref = "https://cdn.example/dl/abc123/movie.mp4"

	res, err := remoteplayback.New().Resolve(context.Background(), v1.PlaybackRequest{Part: remotePart(ref)})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if res.Kind != v1.PlaybackDirect {
		t.Errorf("Kind = %q, want %q", res.Kind, v1.PlaybackDirect)
	}
	if res.URL != ref {
		t.Errorf("URL = %q, want %q", res.URL, ref)
	}
}

// TestResolveMagnetFailsWithTheReason pins the honest failure platform#25 asks
// for. A magnet is the common case for a source with no debrid behind it, and it
// must not silently produce a URL nothing can open: the message has to name the
// missing piece, because that message is what the UI shows a user.
func TestResolveMagnetFailsWithTheReason(t *testing.T) {
	_, err := remoteplayback.New().Resolve(context.Background(),
		v1.PlaybackRequest{Part: remotePart("magnet:?xt=urn:btih:0123456789abcdef")})
	if err == nil {
		t.Fatal("resolving a magnet succeeded; it must fail until a torrent engine or debrid service exists")
	}
	if !strings.Contains(err.Error(), "torrent") {
		t.Errorf("error %q does not say why it failed", err)
	}
}

func TestResolveRejectsUnusableLocations(t *testing.T) {
	cases := []struct {
		name string
		part v1.Part
	}{
		{"local file", v1.Part{Location: v1.MediaLocation{Scheme: v1.LocalLocation, Ref: "/media/movie.mkv"}}},
		{"empty ref", remotePart("")},
		{"not a url", remotePart("some-provider-token")},
		{"scheme with no host", remotePart("http://")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := remoteplayback.New().Resolve(context.Background(), v1.PlaybackRequest{Part: tc.part}); err == nil {
				t.Fatalf("Resolve(%s) succeeded, want an error", tc.name)
			}
		})
	}
}

// TestErrorsDoNotQuoteAWholeCredentialedURL pins the truncation of a quoted
// reference. An unusable reference ends up in an error, errors end up in logs a
// user pastes into an issue, and a signed debrid URL must not travel that way in
// full.
func TestErrorsDoNotQuoteAWholeCredentialedURL(t *testing.T) {
	long := "ftp://cdn.example/" + strings.Repeat("s3cret", 20)

	_, err := remoteplayback.New().Resolve(context.Background(), v1.PlaybackRequest{Part: remotePart(long)})
	if err == nil {
		t.Fatal("Resolve succeeded on an unsupported scheme")
	}
	if strings.Contains(err.Error(), long) {
		t.Error("the error quoted the whole reference; it should be truncated")
	}
}

// TestImportRefuses pins that a consumer module is not a source. Returning an
// empty success would let a mis-routed import look like it worked.
func TestImportRefuses(t *testing.T) {
	if _, err := remoteplayback.New().Import(context.Background(), nil, v1.ImportRequest{}); err == nil {
		t.Fatal("Import succeeded; a playback module materialises nothing and must say so")
	}
}
