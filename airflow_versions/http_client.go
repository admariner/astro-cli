package airflowversions

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/astronomer/astro-cli/pkg/httputil"
)

const (
	RuntimeReleaseURL = "https://updates.astronomer.io/astronomer-runtime"
	AirflowReleaseURL = "https://updates.astronomer.io/astronomer-certified"
)

// Client containers the logger and HTTPClient used to communicate with the HoustonAPI
type Client struct {
	HTTPClient             *httputil.HTTPClient
	useAstronomerCertified bool
}

// NewClient returns a new Client with the logger and HTTP client setup.
func NewClient(c *httputil.HTTPClient, useAstronomerCertified bool) *Client {
	return &Client{
		HTTPClient:             c,
		useAstronomerCertified: useAstronomerCertified,
	}
}

// Request represents empty request
type Request struct{}

// DoWithClient (request) is a wrapper to more easily pass variables to a client.Do request
func (r *Request) DoWithClient(api *Client) (*Response, error) {
	doOpts := &httputil.DoOptions{
		Headers: map[string]string{
			"Accept": "application/json",
		},
	}

	return api.Do(doOpts)
}

// Do executes the given HTTP request and returns the HTTP Response
func (r *Request) Do() (*Response, error) {
	return r.DoWithClient(NewClient(httputil.NewHTTPClient(), false))
}

// Do executes a query against the updates astronomer API, logging out any errors contained in the response object
func (c *Client) Do(doOpts *httputil.DoOptions) (*Response, error) {
	doOpts.Path = RuntimeReleaseURL
	if c.useAstronomerCertified {
		doOpts.Path = AirflowReleaseURL
	}
	doOpts.Method = http.MethodGet
	httpResponse, err := c.HTTPClient.Do(doOpts)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	decode := Response{}
	err = json.NewDecoder(httpResponse.Body).Decode(&decode)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON decode %s response: %w", doOpts.Path, err)
	}

	return &decode, nil
}
