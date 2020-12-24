package cmd

import (
	"context"
	"fmt"
	"github.com/broar/chipmusic-cli/pkg/chipmusic"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/spf13/cobra"
	"time"
)

const (
	defaultTimeout    = 1 * time.Minute
	defaultBufferSize = 1 * time.Second / 10
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

	response, err := client.GetTrack(ctx, trackPageURL)
	if err != nil {
		return fmt.Errorf("failed to download track: %w", err)
	}

	var stream beep.StreamSeekCloser
	var format beep.Format
	switch response.FileType {
	case chipmusic.AudioFileTypeMP3:
		stream, format, err = mp3.Decode(response.Reader)
	default:
		return fmt.Errorf("%s is an unknown audio format", response.FileType)
	}

	if err != nil {
		return fmt.Errorf("failed to decode audio for file format %s: %w", response.FileType, err)
	}

	defer stream.Close()

	if err := speaker.Init(format.SampleRate, format.SampleRate.N(defaultBufferSize)); err != nil {
		return fmt.Errorf("failed to initalize speaker with format %+v: %w", format, err)
	}

	done := make(chan struct{})
	speaker.Play(beep.Seq(stream, beep.Callback(func() {
		done <- struct{}{}
	})))

	<-done

	return nil
}

func play(ctx context.Context, client *chipmusic.Client, trackPageURL string) error {
	track, err := client.GetTrack(ctx, trackPageURL)
	if err != nil {
		return fmt.Errorf("failed to download track: %w", err)
	}

	var stream beep.StreamSeekCloser
	var format beep.Format
	switch track.FileType {
	case chipmusic.AudioFileTypeMP3:
		stream, format, err = mp3.Decode(track.Reader)
	default:
		return fmt.Errorf("%s is an unknown audio format", track.FileType)
	}

	if err != nil {
		return fmt.Errorf("failed to decode audio for file format %s: %w", track.FileType, err)
	}

	defer stream.Close()

	if err := speaker.Init(format.SampleRate, format.SampleRate.N(defaultBufferSize)); err != nil {
		return fmt.Errorf("failed to initalize speaker with format %+v: %w", format, err)
	}

	fmt.Printf("Now playing: %s by %s\n", track.Title, track.Artist)

	done := make(chan struct{})
	speaker.Play(beep.Seq(stream, beep.Callback(func() {
		done <- struct{}{}
	})))

	<-done

	return nil
}