/*
Copyright (c) 2019 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// IMPORTANT: This file has been generated automatically, refrain from modifying it manually as all
// your changes will be lost when the file is generated again.

package v1 // github.com/openshift-online/ocm-sdk-go/authorizations/v1

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/openshift-online/ocm-sdk-go/helpers"
)

// SelfAccessReviewClient is the client of the 'self_access_review' resource.
//
// Manages access review.
type SelfAccessReviewClient struct {
	transport http.RoundTripper
	path      string
	metric    string
}

// NewSelfAccessReviewClient creates a new client for the 'self_access_review'
// resource using the given transport to sned the requests and receive the
// responses.
func NewSelfAccessReviewClient(transport http.RoundTripper, path string, metric string) *SelfAccessReviewClient {
	client := new(SelfAccessReviewClient)
	client.transport = transport
	client.path = path
	client.metric = metric
	return client
}

// Post creates a request for the 'post' method.
//
// Reviews a user's access to a resource
func (c *SelfAccessReviewClient) Post() *SelfAccessReviewPostRequest {
	request := new(SelfAccessReviewPostRequest)
	request.transport = c.transport
	request.path = c.path
	request.metric = c.metric
	return request
}

// SelfAccessReviewPostRequest is the request for the 'post' method.
type SelfAccessReviewPostRequest struct {
	transport http.RoundTripper
	path      string
	metric    string
	query     url.Values
	header    http.Header
	request   *SelfAccessReviewRequest
}

// Parameter adds a query parameter.
func (r *SelfAccessReviewPostRequest) Parameter(name string, value interface{}) *SelfAccessReviewPostRequest {
	helpers.AddValue(&r.query, name, value)
	return r
}

// Header adds a request header.
func (r *SelfAccessReviewPostRequest) Header(name string, value interface{}) *SelfAccessReviewPostRequest {
	helpers.AddHeader(&r.header, name, value)
	return r
}

// Request sets the value of the 'request' parameter.
//
//
func (r *SelfAccessReviewPostRequest) Request(value *SelfAccessReviewRequest) *SelfAccessReviewPostRequest {
	r.request = value
	return r
}

// Send sends this request, waits for the response, and returns it.
//
// This is a potentially lengthy operation, as it requires network communication.
// Consider using a context and the SendContext method.
func (r *SelfAccessReviewPostRequest) Send() (result *SelfAccessReviewPostResponse, err error) {
	return r.SendContext(context.Background())
}

// SendContext sends this request, waits for the response, and returns it.
func (r *SelfAccessReviewPostRequest) SendContext(ctx context.Context) (result *SelfAccessReviewPostResponse, err error) {
	query := helpers.CopyQuery(r.query)
	header := helpers.SetHeader(r.header, r.metric)
	buffer := new(bytes.Buffer)
	err = r.marshal(buffer)
	if err != nil {
		return
	}
	uri := &url.URL{
		Path:     r.path,
		RawQuery: query.Encode(),
	}
	request := &http.Request{
		Method: http.MethodPost,
		URL:    uri,
		Header: header,
		Body:   ioutil.NopCloser(buffer),
	}
	if ctx != nil {
		request = request.WithContext(ctx)
	}
	response, err := r.transport.RoundTrip(request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	result = new(SelfAccessReviewPostResponse)
	result.status = response.StatusCode
	result.header = response.Header
	if result.status >= 400 {
		result.err, err = errors.UnmarshalError(response.Body)
		if err != nil {
			return
		}
		err = result.err
		return
	}
	err = result.unmarshal(response.Body)
	if err != nil {
		return
	}
	return
}

// marshall is the method used internally to marshal requests for the
// 'post' method.
func (r *SelfAccessReviewPostRequest) marshal(writer io.Writer) error {
	var err error
	encoder := json.NewEncoder(writer)
	data, err := r.request.wrap()
	if err != nil {
		return err
	}
	err = encoder.Encode(data)
	return err
}

// SelfAccessReviewPostResponse is the response for the 'post' method.
type SelfAccessReviewPostResponse struct {
	status   int
	header   http.Header
	err      *errors.Error
	response *SelfAccessReviewResponse
}

// Status returns the response status code.
func (r *SelfAccessReviewPostResponse) Status() int {
	return r.status
}

// Header returns header of the response.
func (r *SelfAccessReviewPostResponse) Header() http.Header {
	return r.header
}

// Error returns the response error.
func (r *SelfAccessReviewPostResponse) Error() *errors.Error {
	return r.err
}

// Response returns the value of the 'response' parameter.
//
//
func (r *SelfAccessReviewPostResponse) Response() *SelfAccessReviewResponse {
	if r == nil {
		return nil
	}
	return r.response
}

// GetResponse returns the value of the 'response' parameter and
// a flag indicating if the parameter has a value.
//
//
func (r *SelfAccessReviewPostResponse) GetResponse() (value *SelfAccessReviewResponse, ok bool) {
	ok = r != nil && r.response != nil
	if ok {
		value = r.response
	}
	return
}

// unmarshal is the method used internally to unmarshal responses to the
// 'post' method.
func (r *SelfAccessReviewPostResponse) unmarshal(reader io.Reader) error {
	var err error
	decoder := json.NewDecoder(reader)
	data := new(selfAccessReviewResponseData)
	err = decoder.Decode(data)
	if err != nil {
		return err
	}
	r.response, err = data.unwrap()
	if err != nil {
		return err
	}
	return err
}
