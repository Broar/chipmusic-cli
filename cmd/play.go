package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/broar/chipmusic-cli/pkg/chipmusic"
	"github.com/broar/chipmusic-cli/pkg/player"
	"github.com/spf13/cobra"
	"os"
	"strings"
	"time"
)

const (
	defaultTimeout    = 1 * time.Minute

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

	track, err := client.GetTrack(ctx, trackPageURL)
	if err != nil {
		return fmt.Errorf("failed to download track: %w", err)
	}

	tp, err := player.NewTrackPlayer()
	if err != nil {
		return fmt.Errorf("failed to create track player: %w", err)
	}

	defer tp.Close()

	if err := tp.Play(track); err != nil {
		return fmt.Errorf("failed to play track %s: %w", track.Title, err)
	}

	fmt.Printf("Now playing: %s by %s\n", track.Title, track.Artist)

	go handleTrackControls(tp)

	<-tp.Done()
	return nil
}

func handleTrackControls(tp *player.TrackPlayer) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		command := strings.TrimSpace(scanner.Text())
		if len(command) == 0 {
			continue
		}

		var err error
		switch command {
		case "pause":
			tp.Pause()
		case "stop":
			err = tp.Stop()
		case "loop":
			err = tp.Loop()
		case "skip":
			err = tp.Skip()
		}

		if err != nil {
			fmt.Printf("an error occurred while running the %s command: %v", command, err)
		}
	}
}