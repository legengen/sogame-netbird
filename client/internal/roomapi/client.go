package roomapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	clientnetbird "github.com/legengen/sogame-netbird/client/internal/netbird"
)

const (
	defaultRequestTimeout   = 10 * time.Second
	defaultResponseLimit    = 64 << 10
	createIntentEntropySize = 16
	maximumRetryDelay       = 5 * time.Second
)

var defaultRetryDelays = []time.Duration{
	250 * time.Millisecond,
	500 * time.Millisecond,
	time.Second,
}

type ErrorCode string

const (
	ErrorInvalidRequest     ErrorCode = "invalid_request"
	ErrorRoomUnavailable    ErrorCode = "room_unavailable"
	ErrorOperationConflict  ErrorCode = "operation_conflict"
	ErrorRateLimited        ErrorCode = "rate_limited"
	ErrorServiceUnavailable ErrorCode = "service_unavailable"
	ErrorRequestFailed      ErrorCode = "request_failed"
)

type HTTPError struct {
	StatusCode int
	Code       ErrorCode
	RetryAfter time.Duration
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("Room API request failed with status %d (%s)", e.StatusCode, e.Code)
}

func (e *HTTPError) Transient() bool {
	switch e.StatusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

type ProtocolError struct {
	Reason string
}

func (e *ProtocolError) Error() string {
	return "Room API returned an invalid response: " + e.Reason
}

type Enrollment struct {
	RoomID        string
	RoomCode      string
	ManagementURL string
	SetupKey      *clientnetbird.SetupKey
}

type Peer struct {
	ID        string
	Name      string
	NetBirdIP string
	Connected bool
	Hostname  string
}

type PeerList struct {
	RoomID string
	Peers  []Peer
}

type Client struct {
	baseURL       *url.URL
	httpClient    *http.Client
	timeout       time.Duration
	responseLimit int64
	retryDelays   []time.Duration
	wait          func(context.Context, time.Duration) error
}

var (
	ErrCreateIntentInFlight = errors.New("room create intent is already in flight")
	ErrCreateIntentComplete = errors.New("room create intent is already complete")
)

type CreateIntent struct {
	mu       sync.Mutex
	key      string
	inFlight bool
	complete bool
}

func NewCreateIntent() (*CreateIntent, error) {
	return newCreateIntent(rand.Reader)
}

func newCreateIntent(source io.Reader) (*CreateIntent, error) {
	entropy := make([]byte, createIntentEntropySize)
	if _, err := io.ReadFull(source, entropy); err != nil {
		for index := range entropy {
			entropy[index] = 0
		}
		return nil, errors.New("generate room create intent")
	}
	key := "sogame-room-" + hex.EncodeToString(entropy)
	for index := range entropy {
		entropy[index] = 0
	}
	return &CreateIntent{key: key}, nil
}

func (i *CreateIntent) begin() (string, error) {
	if i == nil {
		return "", errors.New("room create intent is required")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.complete {
		return "", ErrCreateIntentComplete
	}
	if i.inFlight {
		return "", ErrCreateIntentInFlight
	}
	i.inFlight = true
	return i.key, nil
}

func (i *CreateIntent) finish(complete bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.inFlight = false
	if complete {
		i.complete = true
		i.key = ""
	}
}

func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	return newClient(baseURL, httpClient, defaultRequestTimeout, defaultResponseLimit)
}

func newClient(baseURL string, httpClient *http.Client, timeout time.Duration, responseLimit int64) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("invalid Room API base URL")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return nil, errors.New("Room API base URL must use HTTPS")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if timeout <= 0 || responseLimit <= 0 {
		return nil, errors.New("Room API limits must be positive")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	clientCopy := *httpClient
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{
		baseURL:       parsed,
		httpClient:    &clientCopy,
		timeout:       timeout,
		responseLimit: responseLimit,
		retryDelays:   append([]time.Duration(nil), defaultRetryDelays...),
		wait:          waitForRetry,
	}, nil
}

func (c *Client) Create(ctx context.Context, intent *CreateIntent) (Enrollment, error) {
	idempotencyKey, err := intent.begin()
	if err != nil {
		return Enrollment{}, err
	}
	complete := false
	defer func() { intent.finish(complete) }()
	var response enrollmentResponse
	err = c.do(ctx, http.MethodPost, "/rooms", []byte("{}"), map[string]string{
		"Content-Type":    "application/json",
		"Idempotency-Key": idempotencyKey,
	}, &response)
	if err != nil {
		return Enrollment{}, err
	}
	enrollment, err := response.enrollment(true, "")
	if err != nil {
		return Enrollment{}, err
	}
	complete = true
	return enrollment, nil
}

func (c *Client) Join(ctx context.Context, roomCode string) (Enrollment, error) {
	roomCode, err := normalizeRoomCode(roomCode)
	if err != nil {
		return Enrollment{}, err
	}
	body, err := json.Marshal(struct {
		RoomCode string `json:"room_code"`
	}{RoomCode: roomCode})
	if err != nil {
		return Enrollment{}, errors.New("encode room join request")
	}
	var response enrollmentResponse
	if err := c.do(ctx, http.MethodPost, "/rooms/join", body, map[string]string{
		"Content-Type": "application/json",
	}, &response); err != nil {
		return Enrollment{}, err
	}
	return response.enrollment(false, roomCode)
}

func (c *Client) Peers(ctx context.Context, roomCode string) (PeerList, error) {
	roomCode, err := normalizeRoomCode(roomCode)
	if err != nil {
		return PeerList{}, err
	}
	var response peerListResponse
	path := "/rooms/" + roomCode + "/peers"
	if err := c.do(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return PeerList{}, err
	}
	if response.RoomID == "" || response.Peers == nil {
		return PeerList{}, &ProtocolError{Reason: "peer list is incomplete"}
	}
	peers := make([]Peer, 0, len(response.Peers))
	for _, peer := range response.Peers {
		if peer.ID == "" {
			return PeerList{}, &ProtocolError{Reason: "peer identity is missing"}
		}
		peers = append(peers, Peer{
			ID:        peer.ID,
			Name:      peer.Name,
			NetBirdIP: peer.NetBirdIP,
			Connected: peer.Connected,
			Hostname:  peer.Hostname,
		})
	}
	return PeerList{RoomID: response.RoomID, Peers: peers}, nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte, headers map[string]string, target any) error {
	requestCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	for attempt := 0; ; attempt++ {
		err := c.doAttempt(requestCtx, method, path, body, headers, target)
		if err == nil || attempt >= len(c.retryDelays) || !isRetryable(err) {
			return err
		}
		delay := c.retryDelays[attempt]
		var httpError *HTTPError
		if errors.As(err, &httpError) && httpError.RetryAfter > delay {
			delay = httpError.RetryAfter
		}
		if delay > maximumRetryDelay {
			delay = maximumRetryDelay
		}
		if err := c.wait(requestCtx, delay); err != nil {
			return err
		}
	}
}

func (c *Client) doAttempt(ctx context.Context, method, path string, body []byte, headers map[string]string, target any) error {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(c.baseURL.Path, "/") + path
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bodyReader)
	if err != nil {
		return errors.New("build Room API request")
	}
	request.Header.Set("Accept", "application/json")
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &TransportError{}
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, c.responseLimit+1))
	if err != nil {
		return errors.New("read Room API response")
	}
	if int64(len(payload)) > c.responseLimit {
		return &ProtocolError{Reason: "response exceeds size limit"}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return newHTTPError(response.StatusCode, response.Header.Get("Retry-After"))
	}
	if len(bytes.TrimSpace(payload)) == 0 || json.Unmarshal(payload, target) != nil {
		return &ProtocolError{Reason: "response is not valid JSON"}
	}
	return nil
}

type TransportError struct{}

func (*TransportError) Error() string { return "Room API is unavailable" }

func isRetryable(err error) bool {
	var httpError *HTTPError
	if errors.As(err, &httpError) {
		return httpError.Transient()
	}
	var transportError *TransportError
	return errors.As(err, &transportError)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type enrollmentResponse struct {
	RoomID        string `json:"room_id"`
	RoomCode      string `json:"room_code"`
	ManagementURL string `json:"management_url"`
	SetupKey      string `json:"setup_key"`
}

func (r enrollmentResponse) enrollment(requireRoomCode bool, joinedCode string) (Enrollment, error) {
	roomCode := strings.TrimSpace(r.RoomCode)
	if !requireRoomCode {
		roomCode = joinedCode
	}
	if r.RoomID == "" || roomCode == "" || r.ManagementURL == "" || r.SetupKey == "" {
		return Enrollment{}, &ProtocolError{Reason: "enrollment response is incomplete"}
	}
	managementURL, err := url.Parse(r.ManagementURL)
	if err != nil || managementURL.Scheme != "https" || managementURL.Host == "" || managementURL.User != nil {
		return Enrollment{}, &ProtocolError{Reason: "management URL is invalid"}
	}
	return Enrollment{
		RoomID:        r.RoomID,
		RoomCode:      roomCode,
		ManagementURL: managementURL.String(),
		SetupKey:      clientnetbird.NewSetupKey([]byte(r.SetupKey)),
	}, nil
}

type peerListResponse struct {
	RoomID string `json:"room_id"`
	Peers  []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		NetBirdIP string `json:"netbird_ip"`
		Connected bool   `json:"connected"`
		Hostname  string `json:"hostname"`
	} `json:"peers"`
}

func newHTTPError(statusCode int, retryAfter string) error {
	code := ErrorRequestFailed
	switch {
	case statusCode == http.StatusBadRequest:
		code = ErrorInvalidRequest
	case statusCode == http.StatusNotFound:
		code = ErrorRoomUnavailable
	case statusCode == http.StatusConflict:
		code = ErrorOperationConflict
	case statusCode == http.StatusTooManyRequests:
		code = ErrorRateLimited
	case statusCode >= http.StatusInternalServerError:
		code = ErrorServiceUnavailable
	}
	return &HTTPError{
		StatusCode: statusCode,
		Code:       code,
		RetryAfter: parseRetryAfter(retryAfter, time.Now()),
	}
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil && when.After(now) {
		return when.Sub(now)
	}
	return 0
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func normalizeRoomCode(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	parts := strings.Split(value, "-")
	if len(parts) != 3 {
		return "", errors.New("invalid room code")
	}
	for _, part := range parts {
		if len(part) != 4 {
			return "", errors.New("invalid room code")
		}
		for _, character := range part {
			if (character < 'A' || character > 'Z') && (character < '0' || character > '9') {
				return "", errors.New("invalid room code")
			}
		}
	}
	return value, nil
}
