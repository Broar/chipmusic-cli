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
	defaultTrackPageFile = filepath.Join(testDataDir, "sample.html")
)

func TestGetTrack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, err := os.Open("data/sample.html")
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
