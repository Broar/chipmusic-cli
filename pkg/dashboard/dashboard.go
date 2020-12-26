package dashboard

import (
	"errors"
	"fmt"
	"github.com/broar/chipmusic-cli/pkg/chipmusic"
	"github.com/gdamore/tcell/v2"
	"time"
)

const (
	TrackControlPlay  = "play"
	TrackControlPause = "pause"
	TrackControlStop  = "stop"
	TrackControlLoop  = "loop"
	TrackControlSkip  = "skip"

	currentlyPlayingID = "currently-playing"
	trackTimerID       = "time"
)

var (
	// ErrNilTrack is an error returned when attempting to use a nil Screen for a TerminalDashboard
	ErrNilScreen           = errors.New("screen cannot be nil")
	ErrUnknownTrackControl = errors.New("failed to get rune for unknown track control")

	selectedTrackControlStyle = tcell.StyleDefault.Foreground(tcell.ColorReset).Background(tcell.ColorWhite)
	defaultTextStyle          = tcell.StyleDefault.Foreground(tcell.ColorReset).Background(tcell.ColorReset)

	trackControls = []string{
		TrackControlPlay,
		TrackControlPause,
		TrackControlStop,
		TrackControlLoop,
		TrackControlSkip,
	}
)

// TerminalDashboard is a struct capable of displaying an interactive dashboard for playing tracks using a terminal emulator
type TerminalDashboard struct {
	screen   tcell.Screen
	widgets  map[string]*TextWidget
	selected string
	actions  chan string
}

// Option is an alias for a function that modifies a TerminalDashboard. An Option is used to override the default values of TerminalDashboard
type Option func(dashboard *TerminalDashboard) error

// WithScreen allows clients to override the screen used to display the dashboard
func WithScreen(screen tcell.Screen) Option {
	return func(dashboard *TerminalDashboard) error {
		if screen == nil {
			return ErrNilScreen
		}

		dashboard.screen = screen
		return nil
	}
}

// NewTerminalDashboard creates a new TerminalDashboard object that is configured with a list of Options
func NewTerminalDashboard(options ...Option) (*TerminalDashboard, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create default screen: %w", err)
	}

	dashboard := &TerminalDashboard{
		screen: screen,
		widgets: map[string]*TextWidget{
			currentlyPlayingID: NewTextWidget(0, 0, "", defaultTextStyle),
			trackTimerID:       NewTextWidget(0, 2, formatTrackTimer(0, 0), defaultTextStyle),
		},
		selected: TrackControlPlay,
		actions:  make(chan string),
	}

	previous := ""
	x := 0
	for i, trackControl := range trackControls {
		x += len(previous)
		dashboard.widgets[trackControl] = NewTextWidget(x+(i*2), 3, trackControl, defaultTextStyle)
		previous = trackControl
	}

	for _, option := range options {
		if err := option(dashboard); err != nil {
			return nil, err
		}
	}

	return dashboard, nil
}

func (d *TerminalDashboard) Start() error {
	if err := d.init(); err != nil {
		return fmt.Errorf("failed to initalize dashboard: %w", err)
	}

	for {
		d.screen.Show()
		event := d.screen.PollEvent()

		var err error
		switch event := event.(type) {
		case *tcell.EventResize:
			d.screen.Sync()
		case *tcell.EventKey:
			switch event.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlC:
				d.screen.Fini()
				return nil
			case tcell.KeyEnter:
				d.actions <- d.selected
			case tcell.KeyLeft:
				old := d.widgets[d.selected]
				old.SetStyle(defaultTextStyle)
				selected := d.previousTrackControl()
				selected.SetStyle(selectedTrackControlStyle)
				old.Draw(d.screen)
				selected.Draw(d.screen)
			case tcell.KeyRight:
				old := d.widgets[d.selected]
				old.SetStyle(defaultTextStyle)
				selected := d.nextTrackControl()
				selected.SetStyle(selectedTrackControlStyle)
				old.Draw(d.screen)
				selected.Draw(d.screen)
			}
		}

		if err != nil {
			return fmt.Errorf("failed to handle event %+v: %w", event, err)
		}
	}
}

func (d *TerminalDashboard) init() error {
	if err := d.screen.Init(); err != nil {
		return fmt.Errorf("failed to initialize screen: %w", err)
	}

	d.screen.Clear()

	for _, widget := range d.widgets {
		widget.Draw(d.screen)
	}

	return nil
}

func (d *TerminalDashboard) UpdateCurrentTrack(track *chipmusic.Track) {
	if track == nil {
		return
	}

	playing := d.widgets[currentlyPlayingID]
	playing.SetText(fmt.Sprintf("Now playing: %s by %s", track.Title, track.Artist))
	playing.Draw(d.screen)
	d.screen.Show()
}

func (d *TerminalDashboard) UpdateTrackTimer(current, total time.Duration) {
	timer := d.widgets[trackTimerID]
	timer.SetText(formatTrackTimer(current, total))
	timer.Draw(d.screen)
	d.screen.Show()
}

func formatTrackTimer(current, total time.Duration) string {
	return fmt.Sprintf("%s / %s", formatStopwatchTime(current), formatStopwatchTime(total))
}

func formatStopwatchTime(duration time.Duration) string {
	seconds := duration.Round(time.Second).Seconds()
	return fmt.Sprintf("%01d:%02d", int(seconds) / 60, int(seconds) % 60)
}

func (d *TerminalDashboard) nextTrackControl() *TextWidget {
	switch d.selected {
	case TrackControlPlay:
		d.selected = TrackControlPause
	case TrackControlPause:
		d.selected = TrackControlStop
	case TrackControlStop:
		d.selected = TrackControlLoop
	case TrackControlLoop:
		d.selected = TrackControlSkip
	case TrackControlSkip:
		d.selected = TrackControlPlay
	default:
		d.selected = TrackControlPlay
	}

	return d.widgets[d.selected]
}

func (d *TerminalDashboard) previousTrackControl() *TextWidget {
	switch d.selected {
	case TrackControlPlay:
		d.selected = TrackControlSkip
	case TrackControlPause:
		d.selected = TrackControlPlay
	case TrackControlStop:
		d.selected = TrackControlPause
	case TrackControlLoop:
		d.selected = TrackControlStop
	case TrackControlSkip:
		d.selected = TrackControlLoop
	default:
		d.selected = TrackControlPlay
	}

	return d.widgets[d.selected]
}

func (d *TerminalDashboard) Actions() <-chan string {
	return d.actions
}

func (d *TerminalDashboard) Close() error {
	close(d.actions)
	return nil
}
