package ndfc

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"ndfc-collector/pkg/log"

	"github.com/tidwall/gjson"
)

// Client is an HTTP NDFC API client.
// Use ndfc.NewClient to initiate a client.
// This will ensure proper cookie handling and processing of modifiers.
type Client struct {
	// HTTPClient is the *http.Client used for API requests.
	HTTPClient *http.Client
	// host is the NDFC IP or hostname, e.g. 10.0.0.1:80 (port is optional).
	host string
	// Usr is the NDFC username.
	Usr string
	// Pwd is the NDFC password.
	Pwd string
	// LastRefresh is the timestamp of the last token refresh interval.
	LastRefresh time.Time
	// Token is the current authentication token (not used in NDFC, uses session cookies)
	Token string
}

// NewClient creates a new NDFC HTTP client.
// Pass modifiers in to modify the behavior of the client, e.g.
//
//	client, _ := NewClient("ndfc", "user", "password", RequestTimeout(120))
func NewClient(url, usr, pwd string, mods ...func(*Client)) (Client, error) {
	// Normalize the URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	cookieJar, _ := cookiejar.New(nil)
	httpClient := http.Client{
		Timeout:   300 * time.Second,
		Transport: tr,
		Jar:       cookieJar,
	}

	client := Client{
		HTTPClient: &httpClient,
		host:       url,
		Usr:        usr,
		Pwd:        pwd,
	}
	for _, mod := range mods {
		mod(&client)
	}
	return client, nil
}

// NewReq creates a new Req request for this client.
func (client Client) NewReq(method, uri string, body io.Reader, mods ...func(*Req)) Req {
	httpReq, err := http.NewRequest(method, client.host+uri, body)
	if err != nil {
		panic(err)
	}
	req := Req{
		HTTPReq: httpReq,
		Refresh: false, // NDFC uses session cookies, not token refresh
	}
	for _, mod := range mods {
		mod(&req)
	}
	return req
}

// RequestTimeout modifies the HTTP request timeout from the default of 60 seconds.
func RequestTimeout(x time.Duration) func(*Client) {
	return func(client *Client) {
		client.HTTPClient.Timeout = x * time.Second
	}
}

// Do makes a request.
// Requests for Do are built outside of the client, e.g.
//
//	req := client.NewReq("GET", "/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/fabrics", nil)
//	res := client.Do(req)
func (client *Client) Do(req Req) (Res, error) {
	httpRes, err := client.HTTPClient.Do(req.HTTPReq)
	if err != nil {
		return Res{}, err
	}
	defer httpRes.Body.Close()

	body, err := io.ReadAll(httpRes.Body)
	if err != nil {
		return Res{}, errors.New("cannot decode response body")
	}

	res := Res(gjson.ParseBytes(body))

	if httpRes.StatusCode != http.StatusOK {
		return Res{}, fmt.Errorf("received HTTP status %d", httpRes.StatusCode)
	}

	return res, nil
}

// Get makes a GET request and returns a GJSON result.
// Results will be the raw JSON response from NDFC
func (client *Client) Get(path string, mods ...func(*Req)) (Res, error) {
	req := client.NewReq("GET", path, nil, mods...)
	res, err := client.Do(req)
	return res, err
}

// Post makes a POST request and returns a GJSON result.
func (client *Client) Post(path, data string, mods ...func(*Req)) (Res, error) {
	req := client.NewReq("POST", path, strings.NewReader(data), mods...)
	req.HTTPReq.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}

// Login authenticates to NDFC.
func (client *Client) Login() error {
	data := fmt.Sprintf(`{"userName":"%s","userPasswd":"%s","domain":"DefaultAuth"}`,
		client.Usr,
		client.Pwd,
	)
	res, err := client.Post("/login", data, NoRefresh)
	if err != nil {
		return err
	}
	
	// Check if there's an error in the response
	if res.Get("error").Exists() {
		return fmt.Errorf("authentication error: %s", res.Get("error").Str)
	}
	
	client.LastRefresh = time.Now()
	log.Debug().Msg("NDFC authentication successful")
	return nil
}
