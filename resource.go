package airwallex

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
)

// ResponseMetadata describes the HTTP response a resource was decoded from.
// Use RequestID when contacting Airwallex support about a specific call.
type ResponseMetadata struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// RequestID is the Airwallex x-request-id header.
	RequestID string
	// Header holds the response headers.
	Header http.Header
}

// APIResource is embedded in every response type. It preserves the raw JSON
// body of the response, so fields added by newer Airwallex API versions are
// never lost even before this SDK grows typed accessors for them, and
// records which HTTP response the resource came from.
type APIResource struct {
	// Raw is the exact JSON the API returned for this resource.
	Raw json.RawMessage `json:"-"`
	// LastResponse describes the HTTP response this resource was decoded
	// from. For items inside a list it reflects the page's response.
	LastResponse *ResponseMetadata `json:"-"`
}

func (r *APIResource) captureRaw(body []byte) {
	r.Raw = json.RawMessage(append([]byte(nil), body...))
}

func (r *APIResource) captureMeta(meta *ResponseMetadata) {
	r.LastResponse = meta
}

// rawCapturer is implemented by anything embedding APIResource.
type rawCapturer interface {
	captureRaw([]byte)
	captureMeta(*ResponseMetadata)
}

// Params carries fields shared by request-parameter structs. Embed values
// in ExtraParams to send body fields this SDK has no typed field for yet;
// they are merged into the JSON body on top of the typed fields.
type Params struct {
	// ExtraParams is merged into the request body, overriding typed fields
	// on key collision.
	ExtraParams map[string]any `json:"-"`
}

func (p Params) extraParams() map[string]any { return p.ExtraParams }

type extraParamsProvider interface{ extraParams() map[string]any }

// ListParams carries pagination fields shared by all list-parameter structs.
type ListParams struct {
	// PageNum is the 0-based page to start from.
	PageNum int `json:"page_num,omitempty"`
	// PageSize is the number of items per page (server default when 0).
	PageSize int `json:"page_size,omitempty"`
	// ExtraQuery is merged into the query string, for filters this SDK has
	// no typed field for yet.
	ExtraQuery url.Values `json:"-"`
}

func (p ListParams) extraQuery() url.Values { return p.ExtraQuery }

type extraQueryProvider interface{ extraQuery() url.Values }

// newRequestID generates a UUIDv4 for idempotent create calls.
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand never fails on supported platforms; guard anyway.
		panic("airwallex: reading random bytes: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// bodyMap converts a params struct into a JSON-ready map, merging any
// ExtraParams. A nil params yields an empty map.
func bodyMap(params any) (map[string]any, error) {
	body := map[string]any{}
	if params != nil && !isNilPointer(params) {
		encoded, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("airwallex: encoding params: %w", err)
		}
		if err := json.Unmarshal(encoded, &body); err != nil {
			return nil, fmt.Errorf("airwallex: params must encode to a JSON object: %w", err)
		}
		if provider, ok := params.(extraParamsProvider); ok {
			for key, value := range provider.extraParams() {
				body[key] = value
			}
		}
	}
	return body, nil
}

// idempotentBody builds the request body for a money-moving create call,
// generating a request_id when the caller did not supply one. Airwallex
// never executes the same request_id twice, so combined with the SDK
// re-sending identical bytes on retry, creates are idempotent by default.
// The caller's params struct is never mutated.
func idempotentBody(params any) (map[string]any, error) {
	body, err := bodyMap(params)
	if err != nil {
		return nil, err
	}
	if id, ok := body["request_id"].(string); !ok || id == "" {
		body["request_id"] = newRequestID()
	}
	return body, nil
}

// pathEscape percent-encodes a resource id for safe URL-path interpolation.
// It prevents a malicious or malformed id (e.g. "../create") from routing
// the request to a different endpoint.
func pathEscape(id string) string { return url.PathEscape(id) }

// encodeQuery converts a params struct into url.Values using its json tags.
// Zero values are omitted via the struct's omitempty tags; ExtraQuery
// entries are merged on top.
func encodeQuery(params any) (url.Values, error) {
	values := url.Values{}
	if params == nil || isNilPointer(params) {
		return values, nil
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("airwallex: encoding query params: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, fmt.Errorf("airwallex: query params must encode to a JSON object: %w", err)
	}
	for key, raw := range fields {
		values.Set(key, queryValue(raw))
	}
	if provider, ok := params.(extraQueryProvider); ok {
		for key, extra := range provider.extraQuery() {
			for _, value := range extra {
				values.Add(key, value)
			}
		}
	}
	return values, nil
}

// queryValue renders one JSON scalar as its query-string form.
func queryValue(raw json.RawMessage) string {
	var asString string
	if json.Unmarshal(raw, &asString) == nil {
		return asString
	}
	var asNumber float64
	if json.Unmarshal(raw, &asNumber) == nil {
		return strconv.FormatFloat(asNumber, 'f', -1, 64)
	}
	var asBool bool
	if json.Unmarshal(raw, &asBool) == nil {
		return strconv.FormatBool(asBool)
	}
	return string(raw)
}

func isNilPointer(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}
