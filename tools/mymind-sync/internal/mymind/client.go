// Package mymind is a small client for the mymind API (https://access.mymind.com/api).
//
// Every request is authenticated with a freshly minted HS256 JWT whose claims
// pin it to the exact method and path being called.
package mymind

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api.mymind.com"
	DefaultUA      = "zufrieden-mymind-sync/1.0"

	tokenTTL = 5 * time.Minute

	// maxDownloadBytes caps a single file fetched from /objects/:id/blob. Well
	// above anything worth putting on the site, low enough that a runaway
	// response cannot exhaust memory.
	maxDownloadBytes = 64 << 20 // 64 MiB
)

type Client struct {
	baseURL   string
	keyID     string
	secret    []byte
	userAgent string
	http      *http.Client
}

type Option func(*Client)

func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }
func WithUserAgent(ua string) Option       { return func(c *Client) { c.userAgent = ua } }

func New(baseURL, keyID, privateKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(keyID) == "" {
		return nil, fmt.Errorf("mymind: key id is empty")
	}
	if strings.TrimSpace(privateKey) == "" {
		return nil, fmt.Errorf("mymind: private key is empty")
	}
	secret, err := decodeSecret(privateKey)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}
	c := &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		keyID:     strings.TrimSpace(keyID),
		secret:    secret,
		userAgent: DefaultUA,
		http:      &http.Client{Timeout: 60 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// decodeSecret turns the access key secret into HMAC key material. mymind
// hands out base64-encoded random bytes, and it is those *bytes* — not the
// base64 text — that sign the token. Getting this wrong produces a 401
// "Invalid signature", so fail loudly here rather than at request time.
func decodeSecret(privateKey string) ([]byte, error) {
	trimmed := strings.TrimSpace(privateKey)
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		if decoded, err := encoding.DecodeString(trimmed); err == nil {
			return decoded, nil
		}
	}
	return nil, fmt.Errorf("mymind: private key is not valid base64 " +
		"(copy the secret exactly as shown on https://access.mymind.com/extensions)")
}

// sign mints a bearer token scoped to a single method+path.
func (c *Client) sign(method, path string, now time.Time) (string, error) {
	header := map[string]any{
		"alg": "HS256",
		"typ": "JWT",
		"kid": c.keyID,
	}
	claims := map[string]any{
		"path":   path,
		"method": strings.ToUpper(method),
		"iat":    now.Unix(),
		"exp":    now.Add(tokenTTL).Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	enc := base64.RawURLEncoding
	signingInput := enc.EncodeToString(headerJSON) + "." + enc.EncodeToString(claimsJSON)

	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(signingInput))
	return signingInput + "." + enc.EncodeToString(mac.Sum(nil)), nil
}

// APIError carries an RFC 9457 problem document.
type APIError struct {
	Status int
	Title  string
	Detail string
	Body   string
}

func (e *APIError) Error() string {
	msg := e.Title
	if e.Detail != "" {
		if msg != "" {
			msg += ": "
		}
		msg += e.Detail
	}
	if msg == "" {
		msg = strings.TrimSpace(e.Body)
	}
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	return fmt.Sprintf("mymind: HTTP %d: %s", e.Status, msg)
}

// do performs a signed request. path must be the raw request path (no query);
// it is what gets pinned into the token.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(encoded)
	}

	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, target, reqBody)
	if err != nil {
		return nil, err
	}

	token, err := c.sign(method, path, time.Now())
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mymind: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("mymind: reading %s %s: %w", method, path, err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{Status: resp.StatusCode, Body: string(raw)}
		var problem struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
		}
		if json.Unmarshal(raw, &problem) == nil {
			apiErr.Title = problem.Title
			apiErr.Detail = problem.Detail
		}
		return nil, apiErr
	}

	return raw, nil
}

// ListObjects runs a mymind search query and returns the matching objects.
//
// Query syntax reference: https://access.mymind.com/api/search
// e.g. `tag:share -tag:shared`
func (c *Client) ListObjects(ctx context.Context, query string, limit int) ([]Object, error) {
	params := url.Values{}
	if q := strings.TrimSpace(query); q != "" {
		params.Set("q", q)
	}
	// Ask for markdown bodies so anything we lift out of content is ready to publish.
	params.Set("contentAs", "text/markdown")
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	raw, err := c.do(ctx, http.MethodGet, "/objects", params, nil)
	if err != nil {
		return nil, err
	}
	return decodeObjects(raw)
}

// Blob downloads the file behind an object — the image or PDF it was made
// from — and returns the bytes with the MIME type the server reported.
//
// The endpoint answers either with the bytes or with a redirect to mymind's
// media host, where the pre-signed URL is the credential and this key has no
// business being sent.
func (c *Client) Blob(ctx context.Context, objectID string) ([]byte, string, error) {
	if strings.TrimSpace(objectID) == "" {
		return nil, "", fmt.Errorf("mymind: empty object id")
	}
	return c.download(ctx, "/objects/"+url.PathEscape(objectID)+"/blob")
}

// download is do() for an endpoint answering with a file rather than JSON: it
// keeps the response's content type and refuses to read an unbounded body into
// memory.
func (c *Client) download(ctx context.Context, path string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, "", err
	}

	token, err := c.sign(http.MethodGet, path, time.Now())
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "*/*")

	// The token is minted for this one path on the API host; a redirect leads
	// somewhere else, so it is stripped rather than handed to whatever answers
	// there. Go does this itself when the domain changes — doing it explicitly
	// means the guarantee does not rest on where mymind happens to host its
	// media today.
	follow := *c.http
	follow.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after %d redirects", len(via))
		}
		req.Header.Del("Authorization")
		return nil
	}

	resp, err := follow.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("mymind: GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, "", &APIError{Status: resp.StatusCode, Body: string(raw)}
	}

	// One spare byte, so an oversized file is reported rather than silently
	// truncated into a corrupt one.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("mymind: reading GET %s: %w", path, err)
	}
	if len(body) > maxDownloadBytes {
		return nil, "", fmt.Errorf("mymind: GET %s: file is larger than %d bytes", path, maxDownloadBytes)
	}

	contentType, _, _ := strings.Cut(resp.Header.Get("Content-Type"), ";")
	return body, strings.ToLower(strings.TrimSpace(contentType)), nil
}

// AddTags attaches tags to an object. The endpoint is idempotent, so re-tagging
// an already-tagged object is harmless.
func (c *Client) AddTags(ctx context.Context, objectID string, names ...string) error {
	if strings.TrimSpace(objectID) == "" {
		return fmt.Errorf("mymind: empty object id")
	}
	tags := make([]Tag, 0, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			tags = append(tags, Tag{Name: n})
		}
	}
	if len(tags) == 0 {
		return nil
	}
	path := "/objects/" + url.PathEscape(objectID) + "/tags"
	_, err := c.do(ctx, http.MethodPost, path, nil, tags)
	return err
}

// decodeObjects accepts a JSON array, a newline-delimited JSON stream, or an
// enveloped array — the list endpoint can answer with any of them.
func decodeObjects(raw []byte) ([]Object, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil
	}

	switch trimmed[0] {
	case '[':
		var objects []Object
		if err := json.Unmarshal(trimmed, &objects); err != nil {
			return nil, fmt.Errorf("mymind: decoding objects: %w", err)
		}
		return objects, nil

	case '{':
		var envelope struct {
			Objects []Object `json:"objects"`
			Data    []Object `json:"data"`
			Results []Object `json:"results"`
			Items   []Object `json:"items"`
		}
		if json.Unmarshal(trimmed, &envelope) == nil {
			for _, candidate := range [][]Object{envelope.Objects, envelope.Data, envelope.Results, envelope.Items} {
				if len(candidate) > 0 {
					return candidate, nil
				}
			}
		}
		return decodeNDJSON(trimmed)

	default:
		return nil, fmt.Errorf("mymind: unexpected objects payload: %.80s", trimmed)
	}
}

func decodeNDJSON(raw []byte) ([]Object, error) {
	var objects []Object
	dec := json.NewDecoder(bytes.NewReader(raw))
	for {
		var o Object
		if err := dec.Decode(&o); err != nil {
			if err == io.EOF {
				return objects, nil
			}
			return nil, fmt.Errorf("mymind: decoding object stream: %w", err)
		}
		objects = append(objects, o)
	}
}
