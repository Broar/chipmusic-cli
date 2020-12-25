package player

import (
	"errors"
	"github.com/broar/chipmusic-cli/pkg/chipmusic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	testDataDir        = "data"
	defaultTestTimeout = 3 * time.Second
)

var (
	testAudio = filepath.Join(testDataDir, "crickets.mp3")
)

func TestWithBufferSize(t *testing.T) {
	tp, err := NewTrackPlayer(WithBufferSize(-1 * time.Second))
	assert.Error(t, err)
	assert.Nil(t, tp)
}

func TestPlay(t *testing.T) {
	startTrackPlayerTest(t, func(track *chipmusic.Track, tp *TrackPlayer) {
		err := tp.Play(track)
		require.NoError(t, err)
	})
}

func startTrackPlayerTest(t *testing.T, trackPlayerFn func(track *chipmusic.Track, tp *TrackPlayer)) {
	tp, err := NewTrackPlayer()
	require.NoError(t, err)
	require.NotNil(t, tp)

	defer tp.Close()

	file, err := os.Open(testAudio)
	require.NoError(t, err)

	track := &chipmusic.Track{
		Title:    "some.title",
		Artist:   "some.artist",
		FileType: chipmusic.AudioFileTypeMP3,
		Reader:   file,
	}

	defer track.Close()

	go trackPlayerFn(track, tp)

	timer := time.After(defaultTestTimeout)
	for {
		select {
		case <-tp.Done():
			return
		case <-timer:
			t.Logf("track did not finish playing after %s", defaultTestTimeout)
			t.FailNow()
		}
	}
}

func TestPlay_NilTrack(t *testing.T) {
	tp, err := NewTrackPlayer()
	require.NoError(t, err)
	require.NotNil(t, tp)

	err = tp.Play(nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrNilTrack))
}

func TestPlay_BadFileFormat(t *testing.T) {
	tp, err := NewTrackPlayer()
	require.NoError(t, err)
	require.NotNil(t, tp)

	track := &chipmusic.Track{
		Title:    "some.title",
		Artist:   "some.artist",
		FileType: "wav",
	}

	err = tp.Play(track)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnknownFileFormat))
}

func TestPause(t *testing.T) {
	startTrackPlayerTest(t, func(track *chipmusic.Track, tp *TrackPlayer) {
		err := tp.Play(track)
		require.NoError(t, err)

		// Pause, verify the track position never changes, and then unpause
		tp.Pause()
		position := tp.current.Position()
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, position, tp.current.Position())
		tp.Pause()
	})
}

func TestStop(t *testing.T) {
	tp, err := NewTrackPlayer()
	require.NoError(t, err)
	require.NotNil(t, tp)

	startTrackPlayerTest(t, func(track *chipmusic.Track, tp *TrackPlayer) {
		err := tp.Play(track)
		require.NoError(t, err)

		// Stop should rewind the track back to the start and put it into a paused state
		err = tp.Stop()
		assert.NoError(t, err)
		assert.Zero(t, tp.current.Position())
		assert.True(t, tp.ctrl.Paused)

		err = tp.Close()
		require.NoError(t, err)
	})
}

func TestLoop(t *testing.T) {
	tp, err := NewTrackPlayer()
	require.NoError(t, err)
	require.NotNil(t, tp)

	startTrackPlayerTest(t, func(track *chipmusic.Track, tp *TrackPlayer) {
		err := tp.Play(track)
		require.NoError(t, err)

		// Loop and then un-loop
		tp.Loop()
		tp.Loop()
	})
}

// TODO: Test is flaky
func TestSkip(t *testing.T) {
	tp, err := NewTrackPlayer()
	require.NoError(t, err)
	require.NotNil(t, tp)

	startTrackPlayerTest(t, func(track *chipmusic.Track, tp *TrackPlayer) {
		err := tp.Play(track)
		require.NoError(t, err)

		err = tp.Skip()
		assert.NoError(t, err)
		assert.Equal(t, tp.current.Len() - 1, tp.current.Position())
	})
}

func TestAudioControlsWithNoCurrentTrack(t *testing.T) {
	tp, err := NewTrackPlayer()
	require.NoError(t, err)
	require.NotNil(t, tp)

	tp.Pause()
	tp.Loop()
	err = tp.Stop()
	assert.NoError(t, err)
	err = tp.Skip()
	assert.NoError(t, err)
	err = tp.Close()
	assert.NoError(t, err)
}
