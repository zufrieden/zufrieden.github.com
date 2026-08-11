package mymind

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testKeyID = "kid-123"

// The API hands out base64-encoded key material; the raw bytes sign the token.
var (
	testSecretBytes = []byte("0123456789abcdef0123456789abcdef")
	testSecret      = base64.StdEncoding.EncodeToString(testSecretBytes)
)

// decodeToken verifies the HS256 signature and returns the header and claims.
func decodeToken(t *testing.T, authorization string) (map[string]any, map[string]any) {
	t.Helper()

	token, ok := strings.CutPrefix(authorization, "Bearer ")
	if !ok {
		t.Fatalf("Authorization header is not a bearer token: %q", authorization)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected a 3-part JWT, got %d parts", len(parts))
	}

	mac := hmac.New(sha256.New, testSecretBytes)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if parts[2] != want {
		t.Fatalf("bad signature: got %q, want %q", parts[2], want)
	}

	decode := func(segment string) map[string]any {
		raw, err := base64.RawURLEncoding.DecodeString(segment)
		if err != nil {
			t.Fatalf("decoding %q: %v", segment, err)
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("unmarshalling %q: %v", raw, err)
		}
		return out
	}
	return decode(parts[0]), decode(parts[1])
}

func TestListObjectsSignsAndQueries(t *testing.T) {
	var gotPath, gotRawQuery, gotUserAgent string
	var header, claims map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotRawQuery, gotUserAgent = r.URL.Path, r.URL.RawQuery, r.Header.Get("User-Agent")
		header, claims = decodeToken(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `[{"id":"a1","title":"Plain Text","summary":"a summary",
			"source":{"url":"https://example.com/x"},
			"tags":[{"name":"share"}],
			"notes":[{"id":"n1","content":{"type":"text/markdown","body":"my note"}}]}]`)
	}))
	defer server.Close()

	client, err := New(server.URL, testKeyID, testSecret)
	if err != nil {
		t.Fatal(err)
	}

	objects, err := client.ListObjects(context.Background(), "tag:share -tag:shared", 25)
	if err != nil {
		t.Fatal(err)
	}

	if gotPath != "/objects" {
		t.Errorf("path = %q, want /objects", gotPath)
	}
	if gotUserAgent != DefaultUA {
		t.Errorf("User-Agent = %q, want %q", gotUserAgent, DefaultUA)
	}
	for _, want := range []string{"q=tag%3Ashare+-tag%3Ashared", "limit=25", "contentAs=text%2Fmarkdown"} {
		if !strings.Contains(gotRawQuery, want) {
			t.Errorf("query %q missing %q", gotRawQuery, want)
		}
	}

	if header["alg"] != "HS256" || header["kid"] != testKeyID {
		t.Errorf("unexpected JWT header: %v", header)
	}
	// The token is pinned to the path only — the query string is not signed.
	if claims["path"] != "/objects" || claims["method"] != "GET" {
		t.Errorf("unexpected JWT claims: %v", claims)
	}
	iat, exp := claims["iat"].(float64), claims["exp"].(float64)
	if exp-iat != 300 {
		t.Errorf("token lifetime = %vs, want 300s", exp-iat)
	}

	if len(objects) != 1 {
		t.Fatalf("got %d objects, want 1", len(objects))
	}
	obj := objects[0]
	if obj.SourceURL() != "https://example.com/x" {
		t.Errorf("SourceURL = %q", obj.SourceURL())
	}
	if obj.FirstNote() != "my note" {
		t.Errorf("FirstNote = %q", obj.FirstNote())
	}
	if !obj.HasTag("Share") || obj.HasTag("shared") {
		t.Errorf("tag lookup wrong for %v", obj.TagNames())
	}
}

func TestAddTags(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	var claims map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotPath, gotMethod, gotBody = r.URL.Path, r.Method, string(body)
		_, claims = decodeToken(t, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, _ := New(server.URL, testKeyID, testSecret)
	if err := client.AddTags(context.Background(), "obj-1", "shared"); err != nil {
		t.Fatal(err)
	}

	if gotMethod != http.MethodPost || gotPath != "/objects/obj-1/tags" {
		t.Errorf("got %s %s, want POST /objects/obj-1/tags", gotMethod, gotPath)
	}
	if gotBody != `[{"name":"shared"}]` {
		t.Errorf("body = %s", gotBody)
	}
	if claims["path"] != "/objects/obj-1/tags" || claims["method"] != "POST" {
		t.Errorf("unexpected JWT claims: %v", claims)
	}
}

func TestAPIErrorFromProblemJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"title":"Forbidden","detail":"key is read only","status":403}`)
	}))
	defer server.Close()

	client, _ := New(server.URL, testKeyID, testSecret)
	_, err := client.ListObjects(context.Background(), "tag:share", 10)
	if err == nil {
		t.Fatal("expected an error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != 403 {
		t.Errorf("status = %d, want 403", apiErr.Status)
	}
	if !strings.Contains(err.Error(), "key is read only") {
		t.Errorf("error message lost the detail: %v", err)
	}
}

func TestDecodeObjects(t *testing.T) {
	cases := map[string]string{
		"array":    `[{"id":"a"},{"id":"b"}]`,
		"ndjson":   "{\"id\":\"a\"}\n{\"id\":\"b\"}\n",
		"envelope": `{"objects":[{"id":"a"},{"id":"b"}]}`,
	}
	for name, payload := range cases {
		objects, err := decodeObjects([]byte(payload))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(objects) != 2 || objects[0].ID != "a" || objects[1].ID != "b" {
			t.Errorf("%s: got %+v", name, objects)
		}
	}

	if objects, err := decodeObjects([]byte("  ")); err != nil || objects != nil {
		t.Errorf("empty payload: got %v, %v", objects, err)
	}
	if _, err := decodeObjects([]byte("not json")); err == nil {
		t.Error("expected an error for a non-JSON payload")
	}
}

func TestTolerantNoteAndContentShapes(t *testing.T) {
	cases := map[string]string{
		"nested": `{"notes":[{"id":"n1","content":{"type":"text/markdown","body":"hello"}}]}`,
		"string": `{"notes":["hello"]}`,
		"flat":   `{"notes":[{"id":"n1","text":"hello"}]}`,
		"empty1": `{"notes":[{"id":"n1","content":{"body":"  "}},{"content":{"body":"hello"}}]}`,
	}
	for name, payload := range cases {
		var obj Object
		if err := json.Unmarshal([]byte(payload), &obj); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got := obj.FirstNote(); got != "hello" {
			t.Errorf("%s: FirstNote = %q, want %q", name, got, "hello")
		}
	}
}

func TestSourceURLFallsBackToTopLevelURL(t *testing.T) {
	var obj Object
	if err := json.Unmarshal([]byte(`{"url":"https://example.com/top"}`), &obj); err != nil {
		t.Fatal(err)
	}
	if obj.SourceURL() != "https://example.com/top" {
		t.Errorf("SourceURL = %q", obj.SourceURL())
	}
}

func TestCreatedAt(t *testing.T) {
	cases := map[string]struct {
		raw  string
		want string // RFC3339 in UTC, or "" when it should not parse
	}{
		"utc":        {`"2024-03-01T12:00:00Z"`, "2024-03-01T12:00:00Z"},
		"offset":     {`"2024-03-01T14:00:00+02:00"`, "2024-03-01T12:00:00Z"},
		"fractional": {`"2024-03-01T12:00:00.123456Z"`, "2024-03-01T12:00:00Z"},
		"padded":     {`"  2024-03-01T12:00:00Z  "`, "2024-03-01T12:00:00Z"},
		"missing":    {`""`, ""},
		"date only":  {`"2024-03-01"`, ""},
		"epoch":      {`"1709294400"`, ""},
	}
	for name, tc := range cases {
		var obj Object
		if err := json.Unmarshal([]byte(`{"created":`+tc.raw+`}`), &obj); err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		got, ok := obj.CreatedAt()
		if tc.want == "" {
			if ok {
				t.Errorf("%s: CreatedAt = %s, want no timestamp", name, got)
			}
			continue
		}
		if !ok {
			t.Errorf("%s: CreatedAt reported no timestamp, want %s", name, tc.want)
			continue
		}
		if formatted := got.UTC().Format(time.RFC3339); formatted != tc.want {
			t.Errorf("%s: CreatedAt = %s, want %s", name, formatted, tc.want)
		}
	}
}

func TestNewValidatesCredentials(t *testing.T) {
	if _, err := New("", "", testSecret); err == nil {
		t.Error("expected an error for a missing key id")
	}
	if _, err := New("", "kid", ""); err == nil {
		t.Error("expected an error for a missing private key")
	}
	if _, err := New("", "kid", "not valid base64!!"); err == nil {
		t.Error("expected an error for a private key that is not base64")
	}
	client, err := New("", "kid", testSecret)
	if err != nil {
		t.Fatal(err)
	}
	if client.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, DefaultBaseURL)
	}
}

// The signing key is the decoded bytes of the secret, not its base64 text.
// Signing with the text produces a 401 "Invalid signature" from the API.
func TestDecodeSecret(t *testing.T) {
	want := []byte{0xde, 0xad, 0xbe, 0xef}

	for name, encoded := range map[string]string{
		"std padded":    base64.StdEncoding.EncodeToString(want),
		"std unpadded":  base64.RawStdEncoding.EncodeToString(want),
		"url padded":    base64.URLEncoding.EncodeToString(want),
		"url unpadded":  base64.RawURLEncoding.EncodeToString(want),
		"surrounded by": "  " + base64.StdEncoding.EncodeToString(want) + "  ",
	} {
		got, err := decodeSecret(encoded)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: decodeSecret = %x, want %x", name, got, want)
		}
	}

	if _, err := decodeSecret("not valid base64!!"); err == nil {
		t.Error("expected an error for a non-base64 secret")
	}
}
