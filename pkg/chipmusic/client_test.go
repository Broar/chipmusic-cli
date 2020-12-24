package chipmusic

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const (
	testDataDir = "data"
)

var (
	defaultTrackPageFile  = filepath.Join(testDataDir, "track-page.html")
	defaultSearchPageFile = filepath.Join(testDataDir, "search-tracks.html")
)

func TestGetTrack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open(defaultTrackPageFile)
		require.NoError(t, err, "failed to open %s and send as server response", defaultTrackPageFile)

		raw, err := ioutil.ReadAll(file)
		require.NoError(t, err, "failed to read content of %s as server response", defaultTrackPageFile)

		_, err = w.Write(raw)
		require.NoError(t, err, "failed to write %s as server response", defaultTrackPageFile)
	}))

	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err, "failed to create client")

	track, err := client.GetTrack(context.Background(), fmt.Sprintf("%s/some.artist/music/some.music", server.URL))
	require.NoError(t, err, "should not have received an error when getting track")

	defer track.Close()

	assert.Equal(t, "Lovesickness [2a03]", track.Title)
	assert.Equal(t, "Fearofdark", track.Artist)
	assert.NotNil(t, track.Reader)
	assert.Equal(t, AudioFileTypeMP3, track.FileType)
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open(defaultSearchPageFile)
		require.NoError(t, err, "failed to open %s and send as server response", defaultSearchPageFile)

		raw, err := ioutil.ReadAll(file)
		require.NoError(t, err, "failed to read content of %s as server response", defaultSearchPageFile)

		_, err = w.Write(raw)
		require.NoError(t, err, "failed to write %s as server response", defaultSearchPageFile)
	}))

	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err, "failed to create client")

	tracks, err := client.Search(context.Background(), "some.search", TrackFilterRandom, 0)
	assert.NoError(t, err)
	assert.Len(t, tracks, 20)

	expected := []string{
		"https://chipmusic.org/sloopygoop/music/actually-i-want-everything-wario-style-mariah-carey-cover",
		"https://chipmusic.org/Hide+Your+Tigers/music/virtues-lsdj",
		"https://chipmusic.org/daisy/music/bump",
		"https://chipmusic.org/Falling+For+A+Square/music/hope",
		"https://chipmusic.org/Hide+Your+Tigers/music/circuit-circus-lsdj",
		"https://chipmusic.org/theROSSWOODband/music/electric-sheep",
		"https://chipmusic.org/Whitely/music/12",
		"https://chipmusic.org/Whitely/music/careless-whisper---wham!---lsdj-vocal-cover",
		"https://chipmusic.org/Notehead/music/snakerider",
		"https://chipmusic.org/unexpectedbowtie/music/slowly-boiled-frogs",
		"https://chipmusic.org/k7/music/janet-jackson-rockwithu-lsdj-cover",
		"https://chipmusic.org/bloerb/music/jalapeo-peach-panini",
		"https://chipmusic.org/Feryl/music/main-menu",
		"https://chipmusic.org/Dissimulation/music/loop",
		"https://chipmusic.org/Dissimulation/music/test",
		"https://chipmusic.org/amateurlsdj/music/proximity-preview",
		"https://chipmusic.org/vasaturo/music/manic-obsessive-depressive-disorder",
		"https://chipmusic.org/p1xel+sh4der/music/0x00effec7",
		"https://chipmusic.org/ScanianWolf/music/midnight-flight-and-the-dream-you",
		"https://chipmusic.org/Feryl/music/svanholm",
	}

	assert.ElementsMatch(t, expected, tracks)
}
