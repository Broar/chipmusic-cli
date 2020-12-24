package chipmusic

import (
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/atom"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// DefaultBaseURL is redundantly the base URL of the chipmusic.org
	DefaultBaseURL = "https://chipmusic.org"

	// AudioFileTypeMP3 is the expected extension for an MP3 audio file
	AudioFileTypeMP3 AudioFileType = "mp3"

	// TrackFilterNone does not filter for any particular track; instead, it returns the most recently posted tracks
	TrackFilterLatest = "latest"

	// TrackFilterRandom filters for random tracks
	TrackFilterRandom = "random"

	// TrackFilterFeatured filters for featured tracks
	TrackFilterFeatured = "featured"

	// TrackFilterFeatured filters for tracks with high ratings
	TrackFilterHighRatings = "popular"

	defaultTrackFilter = "8"
)

var (
	filters = map[string]string{
		TrackFilterLatest:      defaultTrackFilter,
		TrackFilterRandom:      "8",
		TrackFilterFeatured:    "9",
		TrackFilterHighRatings: "10",
	}
)

// AudioFileType is an enumeration of possible audio file types
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

// Track is song from chipmusic.org. It contains metadata related to the song along with a reader of the track itself
type Track struct {

	// Title is the name of the track
	Title string

	// Artist is the name of the author who composed the track
	Artist string

	// Reader reads the body of the track. This should be closed when a client is finished using a track
	Reader io.ReadCloser

	// FileType represents the type of audio file for this track. This should be used to determine how to interpret and
	// play the content returned from Reader
	FileType AudioFileType
}

func (t *Track) Close() error {
	return t.Reader.Close()
}

// Search performs a search against chipmusic.org, returning a list of URLs to tracks which match. If a search returns
// more tracks than can be returned in a single call, you can use the page parameter to paginate through the additional
// tracks. To iterate through all tracks for a particular search, start with page = 1 and increment it for subsequent
// calls. The order of the tracks returned is undefined. If no tracks are found or there are no other tracks, an empty
// slice is returned
func (c *Client) Search(ctx context.Context, search, filter string, page int) ([]string, error) {
	if page <= 0 {
		page = 1
	}

	resolved, ok := filters[filter]
	if !ok {
		resolved = defaultTrackFilter
	}

	u, err := url.Parse(fmt.Sprintf("%s/music", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	params := url.Values(map[string][]string{
		"#s": {search},
		"p": {strconv.Itoa(page)},
		"f": {resolved},
	})

	u.RawQuery = params.Encode()

	document, err := c.getSearchPageDocument(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get search page document: %w", err)
	}

	return c.parseTracksFromSearch(document), nil
}

func (c *Client) getSearchPageDocument(ctx context.Context, url string) (*goquery.Document, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request to search for tracks: %w", err)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response when searching for tracks: %w", err)
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status code %d when searching for tracks but got %d instead", http.StatusOK, response.StatusCode)
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser when searching for tracks: %w", err)
	}

	return document, nil
}

func (c *Client) parseTracksFromSearch(document *goquery.Document) []string {
	tracks := make([]string, 0, 0)
	links := document.Find("#music_list .item-subject .hn a")
	for _, node := range links.Nodes {
		for _, attribute := range node.Attr {
			if attribute.Key == "href" {
				tracks = append(tracks, attribute.Val)
				break
			}
		}
	}

	return tracks
}

// GetTrack takes a URL to a track page for chipmusic.org and returns a Track. The returned struct contains metadata
// about the track and a reader which can be used to download the track itself for playback. Use FileType in the Track
// to determine how to use the the content returned from the reader
func (c *Client) GetTrack(ctx context.Context, trackPageURL string) (*Track, error) {
	if !strings.HasPrefix(trackPageURL, c.baseURL) {
		return nil, fmt.Errorf("%s is an invalid URL: must start with %s", trackPageURL, c.baseURL)
	}

	document, err := c.getTrackPageDocument(ctx, trackPageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get track page document: %w", err)
	}

	track, err := c.parseTrack(document)
	if err != nil {
		return nil, fmt.Errorf("failed to download track: %w", err)
	}

	return track, nil
}

func (c *Client) getTrackPageDocument(ctx context.Context, trackPageURL string) (*goquery.Document, error) {
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

	return document, nil
}

func (c *Client) parseTrack(document *goquery.Document) (*Track, error) {
	info := document.Find("#item_info")
	if len(info.Nodes) == 0 {
		return nil, fmt.Errorf("failed to find track information")
	}

	track, err := c.parseTrackMetadata(info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse track metadata: %w", err)
	}

	trackDownloadURL, err := parseTrackDownloadURL(info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse track download: %w", err)
	}

	track.FileType = AudioFileType(strings.TrimPrefix(filepath.Ext(trackDownloadURL), "."))

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, trackDownloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get response when downloading track: %w", err)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response when downloading track: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("expected status code %d when downloading track but got %d instead", http.StatusOK, response.StatusCode)
	}

	track.Reader = response.Body

	return track, nil
}

func (c *Client) parseTrackMetadata(info *goquery.Selection) (*Track, error) {
	track := &Track{}
	content := info.Find("#item_content_block")
	if len(content.Nodes) == 0 {
		return nil, errors.New("failed to find track download: expected at least 1 node but found 0")
	}

	for _, node := range content.Children().Nodes {
		if node.DataAtom == atom.Lookup([]byte("h3")) {
			track.Title = node.FirstChild.Data
		}

		if node.DataAtom == atom.Lookup([]byte("span")) {
			child := node.FirstChild
			if child == nil {
				continue
			}

			track.Artist = strings.TrimPrefix(child.FirstChild.Data, "By ")
		}

		if track.Title != "" && track.Artist != "" {
			break
		}
	}

	return track, nil
}

func parseTrackDownloadURL(info *goquery.Selection) (string, error) {
	download := info.Find("#item_play_options #item_download")
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
