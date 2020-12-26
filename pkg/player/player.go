package player

import (
	"context"
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
	// DefaultBufferSize is the default size of the buffer used for the track player
	DefaultBufferSize = 1 * time.Second / 10
	NoCurrentTrack = -1
)

var (
	// ErrNilTrack is an error returned when attempting to play a nil Track
	ErrNilTrack = errors.New("track cannot be nil")

	// ErrUnknownFileFormat is an error returned when a Track's FileFormat cannot be decoded by beep
	ErrUnknownFileFormat = errors.New("unknown file format")
)

// TrackPlayer is a struct capable of playing tracks from readers. It offers a simple suite of audio controls such as
// play, pause, stop, loop, and more.
type TrackPlayer struct {
	bufferSize time.Duration

	mux     sync.Mutex
	ctrl    *beep.Ctrl
	format  beep.Format
	current beep.StreamSeekCloser
	ctx     context.Context
	cancel  context.CancelFunc
	looping bool
}

// Option is an alias for a function that modifies a TrackPlayer. An Option is used to override the default values of TrackPlayer
type Option func(player *TrackPlayer) error

// WithBufferSize allows overriding the buffer size used for playback. Use a lower duration for better responsiveness
// and conversely a higher duration for higher quality but greater CPU usage
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

// Play starts playing a track from its starting position.
//
// This method is not safe to call concurrently. To use this method, clients should do the following:
// 1. Call Play
// 2. Call Done and listen for a signal returned on the channel
// 3. Call Close to release any resources associated with the current track OR simply call Play which already does this
func (t *TrackPlayer) Play(track *chipmusic.Track) error {
	if track == nil {
		return ErrNilTrack
	}

	stream, format, err := t.decodeTrackAudio(track)
	if err != nil {
		return fmt.Errorf("failed to decode track audio: %w", err)
	}

	if err := speaker.Init(format.SampleRate, format.SampleRate.N(t.bufferSize)); err != nil {
		return fmt.Errorf("failed to initalize speaker with format %+v: %w", format, err)
	}

	if err := t.Close(); err != nil {
		return fmt.Errorf("failed to close current track: %w", err)
	}

	t.mux.Lock()

	t.current = stream
	t.format = format
	t.ctrl = &beep.Ctrl{Streamer: stream, Paused: false}
	if t.ctx == nil {
		t.ctx, t.cancel = context.WithCancel(context.Background())
	}

	t.mux.Unlock()

	speaker.Play(beep.Seq(t.ctrl, beep.Callback(func() {
		t.cancel()
	})))

	return nil
}

// Done returns a channel signifying when the current track is done playing which clients can listen on
func (t *TrackPlayer) Done() <-chan struct{} {
	t.mux.Lock()
	defer t.mux.Unlock()
	if t.ctx == nil {
		t.ctx, t.cancel = context.WithCancel(context.Background())
	}

	return t.ctx.Done()
}

func (t *TrackPlayer) decodeTrackAudio(track *chipmusic.Track) (beep.StreamSeekCloser, beep.Format, error) {
	switch track.FileType {
	case chipmusic.AudioFileTypeMP3:
		return mp3.Decode(track.Reader)
	default:
		return beep.StreamSeekCloser(nil), beep.Format{}, fmt.Errorf("%w: %s", ErrUnknownFileFormat, track.FileType)
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
func (t *TrackPlayer) Loop() {
	speaker.Lock()
	defer speaker.Unlock()
	if t.ctrl == nil {
		return
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
}

// Skip seeks to the end of the current track and effectively skips it. If there is no track currently playing,
// this method does nothing
func (t *TrackPlayer) Skip() error {
	speaker.Lock()
	defer speaker.Unlock()
	if t.ctrl == nil {
		return nil
	}

	// Seeking directly the length of the track causes an EOF error to be returned. Seeking to -1 before that position
	// is effectively the same as skipping the entire track and simplifies certain assertions in unit tests. In the
	// future, we can reevaluate if seeking to immediately before the end of the track is necessary
	if err := t.current.Seek(t.current.Len() - 1); err != nil && errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to seek to end of track: %w", err)
	}

	return nil
}

// CurrentTime returns the current position of the track as a duration. If there is no track currently playing, this
// method does nothing
func (t *TrackPlayer) CurrentTime() time.Duration {
	t.mux.Lock()
	defer t.mux.Unlock()
	if t.current == nil {
		return NoCurrentTrack
	}

	speaker.Lock()
	defer speaker.Unlock()
	return t.format.SampleRate.D(t.current.Position())
}

// TotalTime returns the total length of the track as a duration. If there is no track currently playing, this
// method does nothing
func (t *TrackPlayer) TotalTime() time.Duration {
	t.mux.Lock()
	defer t.mux.Unlock()
	if t.current == nil {
		return NoCurrentTrack
	}

	speaker.Lock()
	defer speaker.Unlock()
	return t.format.SampleRate.D(t.current.Len())
}

// Close closes all resources associated with the current track. If there is no track currently playing, this method
// does nothing. This method is implicitly called by Play. There is no need for clients call this method themselves if
// planning to call Play again; however, this method does need to be called when a TrackPlayer will no longer be used
func (t *TrackPlayer) Close() error {
	t.mux.Lock()
	defer t.mux.Unlock()

	if t.current == nil {
		return nil
	}

	if t.ctx != nil {
		t.cancel()
		t.ctx = nil
		t.cancel = nil
	}

	return t.current.Close()
}
