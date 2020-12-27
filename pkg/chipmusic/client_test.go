package chipmusic

import (
	"context"
	"errors"
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

func TestWithBaseURL(t *testing.T) {
	testCases := []struct {
		name string
		url  string
	}{
		{"EmptyURL", ""},
		{"NewlineIsBadURL", "\n"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client, err := NewClient(WithBaseURL(testCase.url))
			assert.Error(t, err)
			assert.Nil(t, client)
		})
	}
}

func TestWithHTTPClient(t *testing.T) {
	client, err := NewClient(WithHTTPClient(nil))
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestWithWorkers(t *testing.T) {
	testCases := []struct {
		name string
		workers  int

	}{
		{"NegativeWorkers", -1},
		{"ZeroWorkers", 0},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client, err := NewClient(WithWorkers(testCase.workers))
			assert.Error(t, err)
			assert.Nil(t, client)
		})
	}
}

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

	client, err := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithWorkers(DefaultWorkers))
	require.NoError(t, err, "failed to create client")

	track, err := client.GetTrack(context.Background(), fmt.Sprintf("%s/some.artist/music/some.music", server.URL))
	require.NoError(t, err, "should not have received an error when getting track")
	assert.Equal(t, "Lovesickness [2a03]", track.Title)
	assert.Equal(t, "Fearofdark", track.Artist)
	assert.NotNil(t, track.Reader)
	assert.Equal(t, AudioFileTypeMP3, track.FileType)
}

func TestGetTrack_NotStatusCodeOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err, "failed to create client")

	track, err := client.GetTrack(context.Background(), fmt.Sprintf("%s/some.artist/music/some.music", server.URL))
	assert.Error(t, err)
	assert.Nil(t, track)
}

func TestGetTrack_ResponseIsNotHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{"some": "json"`))
		require.NoError(t, err, "failed to write server response")
	}))

	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err, "failed to create client")

	track, err := client.GetTrack(context.Background(), fmt.Sprintf("%s/some.artist/music/some.music", server.URL))
	assert.Error(t, err)
	assert.Nil(t, track)
}

func TestGetTrack_ErrorReturnedFromHTTPClient(t *testing.T) {
	httpClient := &http.Client{
		Transport: NewMockTransport(nil, errors.New("an error occurred")),
	}

	client, err := NewClient(WithHTTPClient(httpClient))
	require.NoError(t, err, "failed to create client")

	track, err := client.GetTrack(context.Background(), fmt.Sprintf("%s/some.artist/music/some.music", DefaultBaseURL))
	assert.Error(t, err)
	assert.Nil(t, track)
}

func TestGetTrack_NoURL(t *testing.T) {
	client, err := NewClient()
	require.NoError(t, err, "failed to create client")

	track, err := client.GetTrack(context.Background(), "")
	assert.Error(t, err)
	assert.Nil(t, track)
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

func TestSearch_NotStatusCodeOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))

	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err, "failed to create client")

	tracks, err := client.Search(context.Background(), "some.search", TrackFilterRandom, 0)
	assert.Error(t, err)
	assert.Nil(t, tracks)
}

func TestSearch_ResponseIsNotHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(`{"some": "json"`))
		require.NoError(t, err, "failed to write server response")
	}))

	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	require.NoError(t, err, "failed to create client")

	tracks, err := client.Search(context.Background(), "some.search", TrackFilterRandom, 0)
	assert.NoError(t, err)
	assert.Empty(t, tracks)
}

func TestSearch_ErrorReturnedFromHTTPClient(t *testing.T) {
	httpClient := &http.Client{
		Transport: NewMockTransport(nil, errors.New("an error occurred")),
	}

	client, err := NewClient(WithHTTPClient(httpClient))
	require.NoError(t, err, "failed to create client")

	tracks, err := client.Search(context.Background(), "some.search", TrackFilterRandom, 0)
	assert.Error(t, err)
	assert.Nil(t, tracks)
}

type MockTransport struct {
	response *http.Response
	err      error
}

func NewMockTransport(response *http.Response, err error) *MockTransport {
	return &MockTransport{
		response: response,
		err:      err,
	}
}

func (m *MockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.response, m.err
}