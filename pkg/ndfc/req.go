package ndfc

import (
	"net/http"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Body wraps SJSON for building JSON body strings.
// Usage example:
//
//	Body{}.Set("userName", "admin").Str
type Body struct {
	Str string
}

// Set sets a JSON path to a value.
func (body Body) Set(path, value string) Body {
	res, _ := sjson.Set(body.Str, path, value)
	body.Str = res
	return body
}

// SetRaw sets a JSON path to a raw string value.
// This is primarily used for building up nested structures.
func (body Body) SetRaw(path, rawValue string) Body {
	res, _ := sjson.SetRaw(body.Str, path, rawValue)
	body.Str = res
	return body
}

// Res creates a Res object, i.e. a GJSON result object.
func (body Body) Res() Res {
	return gjson.Parse(body.Str)
}

// Req wraps http.Request for API requests.
type Req struct {
	// HTTPReq is the *http.Request object.
	HTTPReq *http.Request
	// Refresh indicates whether token refresh should be checked for this request.
	// Pass NoRefresh to disable Refresh check.
	Refresh bool
}

// NoRefresh prevents token refresh check.
// Used by the Login method where this would be redundant.
func NoRefresh(req *Req) {
	req.Refresh = false
}

// Query sets an HTTP query parameter.
//
//	client.Get("/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/fabrics", ndfc.Query("filter", "value"))
func Query(k, v string) func(req *Req) {
	return func(req *Req) {
		q := req.HTTPReq.URL.Query()
		q.Add(k, v)
		req.HTTPReq.URL.RawQuery = q.Encode()
	}
}
