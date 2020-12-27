package chipmusic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/atom"
	"golang.org/x/sync/errgroup"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// DefaultBaseURL is redundantly the base URL of the chipmusic.org
	DefaultBaseURL = "https://chipmusic.org"

	DefaultWorkers = 40

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
		TrackFilterLatest:      "0",
		TrackFilterRandom:      defaultTrackFilter,
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

	// workers is the number of goroutines to spin up when downloading a track. This defaults to 10
	workers int
}

// NewClient creates a new Client object that is configured with a list of Options
func NewClient(options ...Option) (*Client, error) {
	client := &Client{
		baseURL: DefaultBaseURL,
		client:  http.DefaultClient,
		workers: DefaultWorkers,
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
		if client == nil {
			return errors.New("client cannot be nil")
		}

		c.client = client
		return nil
	}
}

// WithWorkers allows overriding the default number fo workers used to download a file
func WithWorkers(workers int) Option {
	return func(client *Client) error {
		if workers <= 0 {
			return errors.New("workers must be a positive integer")
		}

		client.workers = workers
		return nil
	}
}

// Track is song from chipmusic.org. It contains metadata related to the song along with a reader of the track itself
type Track struct {

	// Title is the name of the track
	Title string

	// Artist is the name of the author who composed the track
	Artist string

	// Reader reads the body of the track. It is also able to seek to any point within the track
	Reader ReadSeekCloser

	// FileType represents the type of audio file for this track. This should be used to determine how to interpret and
	// play the content returned from Reader
	FileType AudioFileType
}

func (t *Track) Close() error {
	return t.Reader.Close()
}

// ReadSeekCloser is an interface combining the capabilities of ReaderSeeker and Closer. The beep library
type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

// ReadSeekNopCloser is an implementation of ReadSeekCloser which is able to read and seek but closing the reader
// doesn't actually do anything
type ReadSeekNopCloser struct {
	Reader io.ReadSeeker
}

func (r *ReadSeekNopCloser) Read(p []byte) (n int, err error) {
	return r.Reader.Read(p)
}

func (r *ReadSeekNopCloser) Seek(offset int64, whence int) (int64, error) {
	return r.Reader.Seek(offset, whence)
}

func (r *ReadSeekNopCloser) Close() error {
	return nil
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
		"p":  {strconv.Itoa(page)},
		"f":  {resolved},
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
	track := c.parseTrackMetadata(info)
	trackDownloadURL, err := parseTrackDownloadURL(info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse track download: %w", err)
	}

	track.FileType = AudioFileType(strings.TrimPrefix(filepath.Ext(trackDownloadURL), "."))

	request, err := http.NewRequestWithContext(context.Background(), http.MethodHead, trackDownloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get response when downloading track: %w", err)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to get response when downloading track: %w", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected status code %d when downloading track but got %d instead", http.StatusOK, response.StatusCode)
	}

	reader, err := c.downloadTrack(response)
	if err != nil {
		return nil, fmt.Errorf("faild to download track: %w", err)
	}

	track.Reader = &ReadSeekNopCloser{Reader: reader}

	return track, nil
}

func (c *Client) downloadTrack(downloadMetadataResponse *http.Response) (io.ReadSeeker, error) {
	// The server accepts Range requests so we should use them to provide greater throughput
	if downloadMetadataResponse.Header.Get("Accept-Ranges") == "bytes" {
		return c.downloadTrackWithWorkers(downloadMetadataResponse)
	}

	// The server does not accept Range requests so we'll gracefully degrade to a single download request for the whole file
	u := downloadMetadataResponse.Request.URL.String()
	request, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create track download request: %w", err)
	}

	response, err := c.client.Do(request)
	if err != nil {
		return nil,  fmt.Errorf("failed to get response for track download: %w", err)
	}

	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil,  fmt.Errorf("failed to read response for track download: %w", err)
	}

	return bytes.NewReader(content), nil
}

func (c *Client) downloadTrackWithWorkers(downloadMetadataResponse *http.Response) (io.ReadSeeker, error) {
	length, err := strconv.ParseInt(downloadMetadataResponse.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Content-Length header: %w", err)
	}

	// TODO: We can lose some bytes from the division
	content := make([]byte, length, length)
	size := int(length / int64(c.workers))
	group := errgroup.Group{}
	for i := 0; i < c.workers; i++ {
		start := i * size
		end := (i + 1) * size

		// We want to always start with offset of 1 byte so our chunks never overlap except for the first chunk
		if start != 0 {
			start++
		}

		group.Go(func() error {
			u := downloadMetadataResponse.Request.URL.String()
			request, err := http.NewRequest(http.MethodGet, u, nil)
			if err != nil {
				return fmt.Errorf("failed to create track download request: %w", err)
			}

			request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

			response, err := c.client.Do(request)
			if err != nil {
				return fmt.Errorf("failed to get response for track download: %w", err)
			}

			defer response.Body.Close()

			chunk, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return fmt.Errorf("failed to read response for track download: %w", err)
			}

			copy(content[start:start+len(chunk)], chunk)
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("failed to download chunk: %w", err)
	}

	return bytes.NewReader(content), nil
}

func (c *Client) parseTrackMetadata(info *goquery.Selection) *Track {
	track := &Track{}
	content := info.Find("#item_content_block")
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

	return track
}

func parseTrackDownloadURL(info *goquery.Selection) (string, error) {
	download := info.Find("#item_play_options #item_download")
	for _, node := range download.Nodes {
		for _, attribute := range node.Attr {
			if attribute.Key == "href" {
				return attribute.Val, nil
			}
		}
	}

	return "", errors.New("failed to find track download: no URLs found in node attributes")
}
