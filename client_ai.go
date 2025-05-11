// Client code is AI generated and should be reviewed by a human developer before use.
package solidq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// serverResponse is the generic structure for responses from the SolidQ server.
type serverResponse[T any] struct {
	Success  bool           `json:"success"`
	Work     *Work[T]       `json:"work,omitempty"`
	Error    string         `json:"error,omitempty"`
	Count    int            `json:"count,omitempty"`
	Channels map[string]int `json:"channels,omitempty"`
}

// Client is the API client for the SolidQ server.
type Client[T any] struct {
	baseURL         string
	httpClient      *http.Client
	defaultPollWait time.Duration // New field for default poll wait time
}

// Option defines a functional option for configuring the Client.
type Option[T any] func(*Client[T])

// WithHTTPClient allows providing a custom http.Client.
func WithHTTPClient[T any](hc *http.Client) Option[T] {
	return func(c *Client[T]) {
		c.httpClient = hc
	}
}

// WithTimeout sets a timeout for the default http.Client.
// This option is ignored if WithHTTPClient is also used.
func WithTimeout[T any](timeout time.Duration) Option[T] {
	return func(c *Client[T]) {
		// Only override if it's the default or if it's the initial configuration.
		if c.httpClient == http.DefaultClient || (c.httpClient != nil && c.httpClient.Timeout == 0) {
			c.httpClient = &http.Client{Timeout: timeout}
		} else if c.httpClient != nil && timeout > 0 { // Allow overriding existing timeout if a new one is provided
			c.httpClient.Timeout = timeout
		}
	}
}

// WithDefaultPollWait sets the default wait time when the queue is empty in WorkLoop.
func WithDefaultPollWait[T any](duration time.Duration) Option[T] {
	return func(c *Client[T]) {
		if duration > 0 {
			c.defaultPollWait = duration
		}
	}
}

// NewClient creates a new SolidQ API client.
func NewClient[T any](baseURL string, opts ...Option[T]) (*Client[T], error) {
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	c := &Client[T]{
		baseURL:         baseURL,
		httpClient:      http.DefaultClient,
		defaultPollWait: 1 * time.Second, // Default poll wait
	}

	for _, opt := range opts {
		opt(c)
	}
	// Ensure httpClient is not nil if opts cleared it somehow, though current opts don't do that.
	if c.httpClient == nil {
		c.httpClient = http.DefaultClient
	}

	return c, nil
}

// --- Worker Context Definition ---

// WorkerContext provides the current work item and methods to interact with the queue
// to the worker function in WorkLoop.
type WorkerContext[T any] interface {
	// CurrentWork returns the work item being processed.
	CurrentWork() *Work[T]

	// Client returns the underlying SolidQ client for more complex operations if needed,
	// or to access methods not directly exposed on WorkerContext.
	// This gives full access to Push, Count, Reset, ListChannels, etc.
	SolidQClient() *Client[T]

	// Shorthand methods (delegating to SolidQClient)

	// Push adds a new work item to the specified channel.
	Push(channel string, work *Work[T]) error
	// Count retrieves the number of work items in the specified channel.
	Count(channel string) (int, error)
	// Reset clears all work items from the specified channel.
	Reset(channel string) error
	// ListChannels retrieves a map of all channels and their respective work item counts.
	ListChannels() (map[string]int, error)
}

// workerContextImpl implements WorkerContext.
type workerContextImpl[T any] struct {
	work   *Work[T]
	client *Client[T]
}

func newWorkerContext[T any](work *Work[T], client *Client[T]) WorkerContext[T] {
	return &workerContextImpl[T]{
		work:   work,
		client: client,
	}
}

func (wc *workerContextImpl[T]) CurrentWork() *Work[T] {
	return wc.work
}

func (wc *workerContextImpl[T]) SolidQClient() *Client[T] {
	return wc.client
}

func (wc *workerContextImpl[T]) Push(channel string, work *Work[T]) error {
	return wc.client.Push(channel, work)
}

func (wc *workerContextImpl[T]) Count(channel string) (int, error) {
	return wc.client.Count(channel)
}

func (wc *workerContextImpl[T]) Reset(channel string) error {
	return wc.client.Reset(channel)
}

func (wc *workerContextImpl[T]) ListChannels() (map[string]int, error) {
	return wc.client.ListChannels()
}

// --- Private helper methods --- (buildURL, doRequest - assumed to be same as before)
func (c *Client[T]) buildURL(path string, queryParams map[string]string) string {
	base, _ := url.Parse(c.baseURL)
	endpoint, _ := url.Parse(path)
	fullURL := base.ResolveReference(endpoint)

	if queryParams != nil {
		q := fullURL.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		fullURL.RawQuery = q.Encode()
	}
	return fullURL.String()
}

func (c *Client[T]) doRequest(method, urlStr string, body io.Reader) (*serverResponse[T], error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorBodyBytes []byte
		if resp.Body != nil {
			errorBodyBytes, _ = io.ReadAll(resp.Body)
		}
		return nil, fmt.Errorf("server returned non-2xx status: %d %s. Body: %s", resp.StatusCode, http.StatusText(resp.StatusCode), string(errorBodyBytes))
	}

	var sr serverResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		// Attempt to read the body as text for better debugging if JSON decoding fails
		bodyBytes, readErr := io.ReadAll(resp.Body) // This won't work as resp.Body was already read by NewDecoder
		// For robust error, one would read into buffer first then decode.
		// For now, keeping it simple.
		if readErr != nil {
			return nil, fmt.Errorf("failed to decode server response (and failed to read body for debug): %w", err)
		}
		return nil, fmt.Errorf("failed to decode server response: %w. Raw body: %s", err, string(bodyBytes))
	}

	if !sr.Success && sr.Error != "" {
		return &sr, fmt.Errorf("server error: %s", sr.Error)
	}

	return &sr, nil
}

// --- Public API methods --- (Push, Pop, Count, Reset, ListChannels - assumed to be same as before)

func (c *Client[T]) Push(channel string, work *Work[T]) error {
	if channel == "" {
		return fmt.Errorf("channel cannot be empty")
	}
	if work.Id == "" {
		return fmt.Errorf("workID cannot be empty")
	}

	payloadBytes, err := json.Marshal(work.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal payload data: %w", err)
	}

	queryParams := map[string]string{
		"channel": channel,
		"id":      work.Id,
	}
	urlStr := c.buildURL("/solidq/push", queryParams)

	sr, err := c.doRequest(http.MethodPost, urlStr, bytes.NewBuffer(payloadBytes))
	if err != nil {
		if sr != nil && sr.Error != "" {
			return fmt.Errorf("server error on push: %s", sr.Error)
		}
		return fmt.Errorf("push request failed: %w", err)
	}

	if !sr.Success {
		return fmt.Errorf("push operation failed on server without specific error message")
	}
	return nil
}

func (c *Client[T]) Pop(channel string) (*Work[T], error) {
	if channel == "" {
		return nil, fmt.Errorf("channel cannot be empty")
	}

	queryParams := map[string]string{
		"channel": channel,
	}
	urlStr := c.buildURL("/solidq/pop", queryParams)

	sr, err := c.doRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		if sr != nil && sr.Error != "" {
			return nil, fmt.Errorf("server error on pop: %s", sr.Error)
		}
		return nil, fmt.Errorf("pop request failed: %w", err)
	}

	if !sr.Success {
		if sr.Error == "" { // Empty queue
			return nil, nil
		}
		return nil, fmt.Errorf("pop operation failed on server: %s", sr.Error)
	}

	if sr.Work == nil || sr.Work.Id == "" {
		return nil, fmt.Errorf("pop operation succeeded but no work item was returned")
	}
	return sr.Work, nil
}

func (c *Client[T]) Count(channel string) (int, error) {
	if channel == "" {
		return 0, fmt.Errorf("channel cannot be empty")
	}
	queryParams := map[string]string{"channel": channel}
	urlStr := c.buildURL("/solidq/count", queryParams)
	sr, err := c.doRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		if sr != nil && sr.Error != "" {
			return 0, fmt.Errorf("server error on count: %s", sr.Error)
		}
		return 0, fmt.Errorf("count request failed: %w", err)
	}
	if !sr.Success {
		return 0, fmt.Errorf("count operation failed on server: %s", sr.Error)
	}
	return sr.Count, nil
}

func (c *Client[T]) Reset(channel string) error {
	if channel == "" {
		return fmt.Errorf("channel cannot be empty")
	}
	queryParams := map[string]string{"channel": channel}
	urlStr := c.buildURL("/solidq/reset", queryParams)
	sr, err := c.doRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		if sr != nil && sr.Error != "" {
			return fmt.Errorf("server error on reset: %s", sr.Error)
		}
		return fmt.Errorf("reset request failed: %w", err)
	}
	if !sr.Success {
		return fmt.Errorf("reset operation failed on server: %s", sr.Error)
	}
	return nil
}

func (c *Client[T]) ListChannels() (map[string]int, error) {
	urlStr := c.buildURL("/solidq/channels", nil)
	sr, err := c.doRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		if sr != nil && sr.Error != "" {
			return nil, fmt.Errorf("server error on listChannels: %s", sr.Error)
		}
		return nil, fmt.Errorf("listChannels request failed: %w", err)
	}
	if !sr.Success {
		return nil, fmt.Errorf("listChannels operation failed on server: %s", sr.Error)
	}
	if sr.Channels == nil {
		return make(map[string]int), nil
	}
	return sr.Channels, nil
}

// --- WorkLoop Method ---

// WorkLoop continuously polls a channel for work and processes it using the workerFunc.
// It's a blocking call that exits on os.Interrupt or syscall.SIGTERM.
// workerFunc is called synchronously for each piece of work.
// If an error occurs during Pop (not an empty queue), it logs the error and continues.
// If workerFunc itself panics, WorkLoop will also panic. Consider adding panic recovery
// within workerFunc if needed.
//
// The `pollWaitOverride` allows specifying a different poll wait time for this specific loop,
// otherwise the client's default poll wait time is used. Pass 0 to use default.
func (c *Client[T]) WorkLoop(
	channel string,
	workerFunc func(ctx WorkerContext[T]),
	pollWaitOverride time.Duration,
) error {
	if channel == "" {
		return fmt.Errorf("channel cannot be empty for WorkLoop")
	}

	if workerFunc == nil {
		return fmt.Errorf("workerFunc cannot be nil for WorkLoop")
	}

	// Setup graceful shutdown
	// We use a context for shutting down the loop internally when a signal is received.
	loopCtx, cancelLoop := context.WithCancel(context.Background())
	defer cancelLoop() // Ensure cancel is called if WorkLoop exits for other reasons

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan // Wait for signal
		fmt.Printf("\nWorkLoop for channel '%s' received interrupt signal, shutting down...\n", channel)
		cancelLoop() // Signal the loop to stop
	}()

	pollWait := c.defaultPollWait
	if pollWaitOverride > 0 {
		pollWait = pollWaitOverride
	}

	fmt.Printf("Starting WorkLoop for channel '%s'. Polling every %v when empty. Press Ctrl+C to exit.\n", channel, pollWait)

	for {
		select {
		case <-loopCtx.Done(): // Check if shutdown was requested
			fmt.Printf("WorkLoop for channel '%s' stopping due to context cancellation.\n", channel)
			return nil // Exit loop gracefully
		default:
			// Proceed with Pop
		}

		work, err := c.Pop(channel)
		if err != nil {
			// Log Pop error and continue, unless context is cancelled
			// This allows the loop to be resilient to transient network issues.
			select {
			case <-loopCtx.Done():
				// If context was cancelled while Pop was in flight, exit.
				fmt.Printf("WorkLoop for channel '%s' stopping after Pop error due to context cancellation.\n", channel)
				return nil
			default:
				fmt.Printf("Error popping from channel '%s': %v. Retrying after %v.\n", channel, err, pollWait)
				// Wait before retrying after an error, similar to empty queue
				time.Sleep(pollWait) // Use time.Sleep directly here as we are already in the loop.
				continue
			}
		}

		if work != nil {
			// fmt.Printf("WorkLoop on channel '%s' received work: ID=%s\n", channel, work.ID)
			workerCtx := newWorkerContext(work, c)
			// Execute the worker function.
			// Consider adding panic recovery here if workerFunc is untrusted.
			// For now, if workerFunc panics, WorkLoop will panic.
			workerFunc(workerCtx)
			// After workerFunc completes, the loop continues to Pop immediately.
		} else {
			// No work found (queue is empty), wait before polling again
			// fmt.Printf("WorkLoop on channel '%s': Queue empty. Waiting %v.\n", channel, pollWait)
			// We need to ensure this sleep is interruptible by loopCtx.Done()
			select {
			case <-time.After(pollWait):
				// Wait finished, continue loop
			case <-loopCtx.Done():
				fmt.Printf("WorkLoop for channel '%s' stopping during poll wait due to context cancellation.\n", channel)
				return nil // Exit loop gracefully
			}
		}
	}
}
