package cmd

import (
	"context"
	"fmt"
	"github.com/broar/chipmusic-cli/pkg/chipmusic"
	"github.com/broar/chipmusic-cli/pkg/player"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// shuffleCmd represents the shuffle command
var shuffleCmd = &cobra.Command{
	Use:   "shuffle",
	Short: "Play a shuffle of songs from chipmusic.org",
	Run: func(cmd *cobra.Command, args []string) {
		if err := shuffle(); err != nil {
			panic(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(shuffleCmd)
	shuffleCmd.Flags().String("search", "", "Add search text to the shuffle to limit results")
	shuffleCmd.Flags().String("filter", "", "Set a filter for the shuffle. Allowed filters: [latest, random, featured, popular]")
}

func shuffle() error {
	client, err := chipmusic.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create chipmusic client: %w", err)
	}

	tp, err := player.NewTrackPlayer()
	if err != nil {
		return fmt.Errorf("failed to create track player: %w", err)
	}

	defer tp.Close()

	go handleTrackControls(tp)

	var tracks []string
	page := 1
	for {
		err, done := getAndPlayTracks(tracks, page, client, tp)
		if err != nil {
			return fmt.Errorf("failed to play tracks: %w", err)
		}

		if done {
			return nil
		}

		page++
	}
}

func getAndPlayTracks(tracks []string, page int, client *chipmusic.Client, tp *player.TrackPlayer) (error, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	tracks, err := client.Search(ctx, viper.GetString("search"), viper.GetString("filter"), page)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to download track: %w", err), false
	}

	if len(tracks) == 0 {
		return nil, true
	}

	for _, trackURL := range tracks {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		track, err := client.GetTrack(ctx, trackURL)
		if err != nil {
			cancel()
			return fmt.Errorf("failed to download track: %w", err), false
		}

		cancel()

		if err := tp.Play(track); err != nil {
			return fmt.Errorf("failed to play track %s: %w", track.Title, err), false
		}

		fmt.Printf("Now playing: %s by %s\n", track.Title, track.Artist)

		<-tp.Done()
	}

	return nil, false
}
