package bean

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type (
	Context interface {
		context.Context

		// Request returns `*http.Request`.
		Request() *http.Request

		// SetRequest sets `*http.Request`.
		SetRequest(r *http.Request)

		// Response returns `http.ResponseWriter`.
		Response() http.ResponseWriter

		// Get returns the value for the given key string from the context.
		// If the value doesn't exist it returns (nil, false).
		Get(key string) (any, bool)

		MustGet(key string) any

		// Set saves data in the context.
		Set(key string, val any)

		Keys() map[string]any

		// Param returns path parameter by name.
		Param(name string) string

		// AddParam adds param to context and
		AddParam(name, value string)

		// Query returns the query param for the provided name.
		Query(name string) string

		// QueryParams returns the query parameters as `url.Values`.
		QueryParams() url.Values

		// Bind binds the request body into provided type `i`. The default binder
		// does it based on Content-Type header.
		Bind(i any, _ Context) error

		// Validate validates provided `i`. It is usually called after `Context#Bind()`.
		// Validator must be registered using `Echo#Validator`.
		Validate(i any) error

		// Render renders a template with data and sends a text/html response with status
		// code. Renderer must be registered using `Echo.Renderer`.
		Render(code int, name string, data any) error

		// HTML sends an HTTP response with status code.
		HTML(code int, html string) error

		// HTMLBlob sends an HTTP blob response with status code.
		HTMLBlob(code int, b []byte) error

		// String sends a string response with status code.
		String(code int, s string) error

		// JSON sends a JSON response with status code.
		JSON(code int, i any) error

		// Error invokes the registered HTTP error handler. Generally used by middleware.
		Error(err error)

		// Reset resets the context after request completes. It must be called along
		// with `Echo#AcquireContext()` and `Echo#ReleaseContext()`.
		// See `Echo#ServeHTTP()`
		Reset(r *http.Request, w http.ResponseWriter)

		// Cookie returns the named cookie provided in the request.
		Cookie(name string) (*http.Cookie, error)

		// SetCookie adds a `Set-Cookie` header in HTTP response.
		SetCookie(cookie *http.Cookie)

		// Cookies returns the HTTP cookies sent with the request.
		Cookies() []*http.Cookie
	}

	Binder interface {
		Bind(i interface{}) error
	}

	// Validator is the interface that wraps the Validate function.
	Validator interface {
		Validate(i interface{}) error
	}

	beanContext struct {
		request   *http.Request
		response  http.ResponseWriter
		mu        sync.RWMutex
		keys      map[string]any
		binder    Binder
		validator Validator
		params    [][2]string
		query     url.Values
	}

	HandlerFunc    func(c Context) error
	MiddlewareFunc func(HandlerFunc) HandlerFunc
)

const (
	defaultIndent = "  "
)

var (
	// beanContext implement the Context interface
	_ Context = (*beanContext)(nil)
)

func (bc *beanContext) Request() *http.Request {
	return bc.request
}

func (bc *beanContext) SetRequest(r *http.Request) {
	bc.request = r
}

func (bc *beanContext) Response() http.ResponseWriter {
	return bc.response
}

func (bc *beanContext) Keys() map[string]any {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.keys
}

func (bc *beanContext) MustGet(key string) any {
	val, e := bc.Get(key)
	if !e {
		panic(fmt.Sprintf("beanContext: %q not exist", key))
	}
	return val
}

// Get returns the value for the given key string from the context.
// If the value doesn't exist it returns (nil, false).
func (bc *beanContext) Get(key string) (value any, e bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	value, e = bc.keys[key]
	return
}

// Set is saving a new key-value pair exclusively for this context.
// It also initializes `bc.keys` if it was not initialized previously.
func (bc *beanContext) Set(key string, value any) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if bc.keys == nil {
		bc.keys = make(map[string]any)
	}

	bc.keys[key] = value
}

func (bc *beanContext) Param(name string) string {
	for _, data := range bc.params {
		if data[0] == name {
			return data[1]
		}
	}
	return ""
}

func (bc *beanContext) AddParam(name, value string) {
	bc.params = append(bc.params, [2]string{name, value})
}

func (bc *beanContext) Query(name string) string {
	return bc.QueryParams().Get(name)
}

func (bc *beanContext) QueryParams() url.Values {
	if bc.query == nil {
		bc.query = bc.request.URL.Query()
	}
	return bc.query
}

func (bc *beanContext) Bind(i any, _ Context) error {
	return bc.binder.Bind(i)
}

func (bc *beanContext) SetBinder(binder Binder) {
	bc.binder = binder
}

func (bc *beanContext) Validate(i any) error {
	if bc.validator != nil {
		return bc.validator.Validate(i)
	}
	return nil
}

func (bc *beanContext) Render(code int, name string, data any) error {
	// TODO implement me
	panic("implement me")
}

func (bc *beanContext) HTML(code int, html string) error {
	// TODO implement me
	panic("implement me")
}

func (bc *beanContext) HTMLBlob(code int, b []byte) error {
	// TODO implement me
	panic("implement me")
}

func (bc *beanContext) String(code int, s string) error {
	// TODO implement me
	panic("implement me")
}

func (bc *beanContext) JSON(code int, i any) error {
	indent := ""
	if _, pretty := bc.QueryParams()["pretty"]; pretty {
		indent = defaultIndent
	}
	return bc.json(code, i, indent)
}

func (bc *beanContext) json(code int, i interface{}, indent string) error {
	bc.writeContentType("application/json;charset=UTF-8")
	bc.response.WriteHeader(code)
	enc := json.NewEncoder(bc.response)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	return enc.Encode(i)
}

func (bc *beanContext) writeContentType(value string) {
	header := bc.response.Header()
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", value)
	}
}

func (bc *beanContext) Error(err error) {
	// TODO implement me
	panic("implement me")
}

func (bc *beanContext) Reset(r *http.Request, w http.ResponseWriter) {
	bc.request = r
	bc.response = w
}

func (bc *beanContext) Cookie(name string) (*http.Cookie, error) {
	return bc.request.Cookie(name)
}

func (bc *beanContext) SetCookie(cookie *http.Cookie) {
	http.SetCookie(bc.Response(), cookie)
}

func (bc *beanContext) Cookies() []*http.Cookie {
	return bc.request.Cookies()
}

// Deadline returns that there is no deadline (ok==false) when c.Request has no Context.
func (bc *beanContext) Deadline() (deadline time.Time, ok bool) {
	if bc.request == nil {
		return
	}
	return bc.request.Context().Deadline()
}

// Done returns nil (chan which will wait forever) when c.Request has no Context.
func (bc *beanContext) Done() <-chan struct{} {
	if bc.request == nil {
		return nil
	}
	return bc.request.Context().Done()
}

// Err returns nil when c.Request has no Context.
func (bc *beanContext) Err() error {
	if bc.request == nil {
		return nil
	}
	return bc.request.Context().Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (bc *beanContext) Value(key any) any {
	if key == 0 {
		return bc.request
	}

	if keyAsString, ok := key.(string); ok {
		if val, exists := bc.Get(keyAsString); exists {
			return val
		}
	}
	if bc.request == nil {
		return nil
	}
	return bc.request.Context().Value(key)
}
