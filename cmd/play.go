package cmd

import (
	"context"
	"fmt"
	"github.com/broar/chipmusic-cli/pkg/chipmusic"
	"github.com/broar/chipmusic-cli/pkg/dashboard"
	"github.com/broar/chipmusic-cli/pkg/player"
	"github.com/spf13/cobra"
	"time"
)

const (
	defaultTimeout = 1 * time.Minute
)

var playCmd = &cobra.Command{
	Use:   "play track",
	Short: "Play a track with an exact URL from chipmusic.org",
	Run: func(cmd *cobra.Command, args []string) {
		if err := playTrack(args[0]); err != nil {
			panic(err)
		}
	},
	Args: cobra.ExactArgs(1),
}

func init() {
	rootCmd.AddCommand(playCmd)
}

func playTrack(trackPageURL string) error {
	client, err := chipmusic.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create chipmusic client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tp, err := player.NewTrackPlayer()
	if err != nil {
		return fmt.Errorf("failed to create track player: %w", err)
	}

	defer tp.Close()

	db, err := dashboard.NewTerminalDashboard()
	if err != nil {
		return fmt.Errorf("failed to create terminal dashboard: %w", err)
	}

	defer db.Close()

	actions := db.Actions()
	go func() {
		if err := db.Start(); err != nil {
			panic(err)
		}
	}()

	go func() {
		handleTrackControlActions(actions, tp)
	}()

	track, err := client.GetTrack(ctx, trackPageURL)
	if err != nil {
		return fmt.Errorf("failed to download track: %w", err)
	}

	db.UpdateCurrentlyPlayingTrack(track)

	if err := tp.Play(track); err != nil {
		return fmt.Errorf("failed to play track %s: %w", track.Title, err)
	}

	<-tp.Done()
	return nil
}

func handleTrackControlActions(actions <-chan string, tp *player.TrackPlayer) {
	for {
		select {
		case action := <-actions:
			var err error
			switch action {
			case dashboard.TrackControlPlay:
				// Nothing to do
			case dashboard.TrackControlPause:
				tp.Pause()
			case dashboard.TrackControlStop:
				err = tp.Stop()
			case dashboard.TrackControlLoop:
				tp.Loop()
			case dashboard.TrackControlSkip:
				err = tp.Skip()
			default:
				fmt.Printf("received unknown track control: %v\n", action)
			}

			if err != nil {
				fmt.Printf("failed to handle track control: %v: %v\n", action, err)
			}
		}
	}
}