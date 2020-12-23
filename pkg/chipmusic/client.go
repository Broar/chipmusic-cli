package chipmusic

import (
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"net/url"
)

const (
	// DefaultBaseURL is redundantly the base URL of the chipmusic.org
	DefaultBaseURL = "https://chipmusic.org"

	AudioFileTypeMP3 AudioFileType = "mp3"
)

type AudioFileType string

// Client is a struct capable of interacting with chipmusic.org
type Client struct {
	// baseURL is the base URL of the chipmusic.org forums. This defaults to DefaultBaseURL
	baseURL string

	// client is the HTTP client used to make requests. This defaults to http.DefaultClient
	client *http.Client
}

// NewClient creates a new Client object that is configured with a list of Options
func NewClient(options ...Option) (*Client, error) {
	client := &Client{
		baseURL: DefaultBaseURL,
		client:  http.DefaultClient,
	}

	for _, option := range options {
		if err := option(client); err != nil {
			return nil, fmt.Errorf("failed to create client: %v", err)
		}
	}

	return client, nil
}

// Option is an alias for a function that modifies a Client. An Option is used to override the default values of Client
type Option func(*Client) error

// WithBaseURL allows overriding the base URL for chipmusic.org
func WithBaseURL(baseURL string) Option {
	return func(c *Client) error {
		if baseURL == "" {
			return errors.New("URL cannot be empty")
		}

		if _, err := url.Parse(baseURL); err != nil {
			return fmt.Errorf("failed to parse base URL: %w", err)
		}

		c.baseURL = baseURL
		return nil
	}
}

// WithHTTPClient allows overriding the default HTTP client used to make requests
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		if c == nil {
			return errors.New("client cannot be nil")
		}

		c.client = client
		return nil
	}
}

type GetTrackResponse struct {
	Title  string
	Artist string
	Rating int
	Views  int
	Track  io.ReadCloser
	FileType AudioFileType
}

func (c *Client) GetTrack(ctx context.Context, trackPageURL string) (*GetTrackResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, trackPageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request to get track page: %w", err)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response when getting track page: %w", err)
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status code %d when getting track page but got %d instead", http.StatusOK, response.StatusCode)
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser when getting track page: %w", err)
	}

	resp, err := c.parseTrackMetadata(ctx, document)

	trackDownloadURL, err := parseTrackDownloadURL(document)
	if err != nil {
		return nil, fmt.Errorf("failed to parse track download: %w", err)
	}

	// TODO: Using the original context with a timeout ends the download too early
	request, err = http.NewRequestWithContext(context.Background(), http.MethodGet, trackDownloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get response when downloading track: %w", err)
	}

	response, err = c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response when downloading track: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("expected status code %d when downloading track but got %d instead", http.StatusOK, response.StatusCode)
	}

	resp.Track = response.Body

	// TODO
	resp.FileType = AudioFileTypeMP3

	return resp, nil
}

func (c *Client) parseTrackMetadata(ctx context.Context, document *goquery.Document) (*GetTrackResponse, error) {
	resp := &GetTrackResponse{}

	// TODO

	rating, err := c.getTrackRating(ctx, resp.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to get track rating: %w", err)
	}

	resp.Rating = rating

	return resp, nil
}

func (c *Client) getTrackRating(ctx context.Context, title string) (int, error) {
	return 0, nil
}

func parseTrackDownloadURL(document *goquery.Document) (string, error) {
	download := document.Find("#item_info #item_play_options #item_download")
	if download == nil {
		return "", errors.New("failed to find track download")
	}

	if len(download.Nodes) == 0 {
		return "", errors.New("failed to find track download: expected at least 1 node but found 0")
	}

	node := download.Nodes[0]
	for _, attribute := range node.Attr {
		if attribute.Key == "href" {
			return attribute.Val, nil
		}
	}

	return "", errors.New("failed to find track download: no URLs found in node attributes")
}


