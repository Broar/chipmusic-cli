package player

import (
	"errors"
	"fmt"
	"github.com/broar/chipmusic-cli/pkg/chipmusic"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"io"
	"math"
	"sync"
	"time"
)

const (
	// DefaultBufferSize is the default size of the buffer used for the track player. Use a lower duration for better
	// responsiveness and conversely a higher duration for higher quality but greater CPU usage
	DefaultBufferSize = 1 * time.Second / 10
)

// TrackPlayer is a struct capable of playing tracks from readers. It offers a simple suite of audio controls such as
// play, pause, stop, loop, and more.
type TrackPlayer struct {
	bufferSize time.Duration

	mux     sync.Mutex
	ctrl    *beep.Ctrl
	current beep.StreamSeekCloser
	looping bool
	done    chan struct{}
}

// Option is an alias for a function that modifies a TrackPlayer. An Option is used to override the default values of TrackPlayer
type Option func(player *TrackPlayer) error

// WithBufferSize allows overriding the buffer size used for playback
func WithBufferSize(bufferSize time.Duration) Option {
	return func(player *TrackPlayer) error {
		if bufferSize <= 0 {
			return errors.New("buffer size must be greater than 0")
		}

		player.bufferSize = bufferSize
		return nil
	}
}

// NewTrackPlayer creates a new TrackPlayer object that is configured with a list of Options
func NewTrackPlayer(options ...Option) (*TrackPlayer, error) {
	player := &TrackPlayer{
		bufferSize: DefaultBufferSize,
		mux:        sync.Mutex{},
	}

	for _, option := range options {
		if err := option(player); err != nil {
			return nil, err
		}
	}

	return player, nil
}

// Play starts playing a track from its starting position. Clients can call this method any number of times. If there
// is a currently loaded track, any resources associated with it will be automatically closed
func (t *TrackPlayer) Play(track *chipmusic.Track) error {
	stream, format, err := t.decodeTrackAudio(track)
	if err != nil {
		return fmt.Errorf("failed to decode track audio: %w", err)
	}

	if err := speaker.Init(format.SampleRate, format.SampleRate.N(t.bufferSize)); err != nil {
		return fmt.Errorf("failed to initalize speaker with format %+v: %w", format, err)
	}

	t.mux.Lock()
	defer t.mux.Unlock()

	if err := t.Close(); err != nil {
		return fmt.Errorf("failed to close current track: %w", err)
	}

	t.current = stream
	t.ctrl = &beep.Ctrl{Streamer: stream, Paused: false}
	t.done = make(chan struct{})
	speaker.Play(beep.Seq(t.ctrl, beep.Callback(func() {
		t.done <- struct{}{}
	})))

	return nil
}

// Done returns a channel signifying when the current track is done playing which clients can listen on
func (t *TrackPlayer) Done() <-chan struct{} {
	return t.done
}

func (t *TrackPlayer) decodeTrackAudio(track *chipmusic.Track) (beep.StreamSeekCloser, beep.Format, error) {
	switch track.FileType {
	case chipmusic.AudioFileTypeMP3:
		return mp3.Decode(track.Reader)
	default:
		return beep.StreamSeekCloser(nil), beep.Format{}, fmt.Errorf("%s is an unknown audio format", track.FileType)
	}
}

// Pause pauses/unpauses the currently playing track. If there is no track is currently playing, this method does nothing
func (t *TrackPlayer) Pause() {
	speaker.Lock()
	defer speaker.Unlock()
	if t.ctrl == nil {
		return
	}

	t.ctrl.Paused = !t.ctrl.Paused
}

// Stop pauses the currently playing track and resets its position to the start. If there is no track currently playing,
// this method does nothing
func (t *TrackPlayer) Stop() error {
	speaker.Lock()
	defer speaker.Unlock()
	if t.ctrl == nil {
		return nil
	}

	t.ctrl.Paused = true
	if err := t.current.Seek(0); err != nil {
		return fmt.Errorf("failed to seek to start of track: %w", err)
	}

	return nil
}

// Loop loops the currently playing track. If the current track is already looping, this method disables looping. If
// there is no track currently playing, this method does nothing
func (t *TrackPlayer) Loop() error {
	speaker.Lock()
	defer speaker.Unlock()
	if t.ctrl == nil {
		return nil
	}

	t.mux.Lock()
	defer t.mux.Unlock()

	if t.looping {
		t.ctrl.Streamer = t.current
		t.looping = false
	} else {
		t.ctrl.Streamer = beep.Loop(math.MaxInt32, t.current)
		t.looping = true
	}

	return nil
}

// Skip seeks to the end of the current track and effectively skips it. If there is no track currently playing,
// this method does nothing
func (t *TrackPlayer) Skip() error {
	speaker.Lock()
	defer speaker.Unlock()
	if t.ctrl == nil {
		return nil
	}

	if err := t.current.Seek(t.current.Len()); err != nil && errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to seek to end of track: %w", err)
	}

	return nil
}

func (t *TrackPlayer) Close() error {
	if t.current != nil {
		return t.current.Close()
	}

	return nil
}
