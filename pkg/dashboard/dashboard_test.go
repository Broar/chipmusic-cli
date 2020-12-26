package dashboard

import (
	"github.com/broar/chipmusic-cli/pkg/chipmusic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTerminalDashboard_UpdateCurrentTrack(t *testing.T) {
	testCases := []struct {
		name     string
		track    *chipmusic.Track
		expected string
	}{
		{"NilTrack", nil, ""},
		{"Defaults", &chipmusic.Track{}, "Now playing:  by "},
		{"AllFieldsSet", &chipmusic.Track{Title: "some.title", Artist: "some.artist"}, "Now playing: some.title by some.artist"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			db, err := NewTerminalDashboard(WithScreen(&MockScreen{}))
			require.NoError(tt, err)

			defer db.Close()

			db.UpdateCurrentTrack(testCase.track)
			widget, ok := db.widgets[currentlyPlayingID]
			require.True(tt, ok)

			assert.Equal(tt, []string{testCase.expected}, widget.base.drawing)
		})
	}
}

func TestTerminalDashboard_UpdateTrackTimer(t *testing.T) {
	testCases := []struct {
		name     string
		current  time.Duration
		total    time.Duration
		expected string
	}{
		{"Defaults", 0, 0, "0:00 / 0:00"},
		{"LessThanSecondRoundsHalfUp", 499 * time.Millisecond, 500 * time.Millisecond, "0:00 / 0:01"},
		{"OnlyTotal", 0, 1 * time.Second, "0:00 / 0:01"},
		{"OnlyCurrent", 1 * time.Second, 0, "0:01 / 0:00"},
		{"ExactlyOneMinute", 60 * time.Second, 1 * time.Minute, "1:00 / 1:00"},
		{"GreaterThanOneMinute", 75 * time.Second, 75 * time.Second, "1:15 / 1:15"},
		{"DoubleDigits", 10 * time.Minute, 10 * time.Minute, "10:00 / 10:00"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			db, err := NewTerminalDashboard(WithScreen(&MockScreen{}))
			require.NoError(tt, err)

			defer db.Close()

			db.UpdateTrackTimer(testCase.current, testCase.total)
			widget, ok := db.widgets[trackTimerID]
			require.True(tt, ok)

			assert.Equal(tt, []string{testCase.expected}, widget.base.drawing)
		})
	}
}

func TestTerminalDashboard_Start(t *testing.T) {

}
