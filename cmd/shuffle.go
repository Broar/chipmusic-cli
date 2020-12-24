package cmd

import (
	"context"
	"fmt"
	"github.com/broar/chipmusic-cli/pkg/chipmusic"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

// shuffleCmd represents the shuffle command
var shuffleCmd = &cobra.Command{
	Use:   "shuffle",
	Short: "Shuffle",
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

	var tracks []string
	page := 1
	for {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		tracks, err = client.Search(ctx, viper.GetString("search"), viper.GetString("filter"), page)
		if err != nil {
			cancel()
			return fmt.Errorf("failed to download track: %w", err)
		}

		cancel()

		if len(tracks) == 0 {
			break
		}

		for _, trackURL := range tracks {
			ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
			if err := play(ctx, client, trackURL); err != nil {
				cancel()
				return fmt.Errorf("failed to play track: %w", err)
			}

			cancel()
		}
	}

	return nil
}
