/*
Copyright 2021 The Kubernetes Authors.

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

package rest

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"k8s.io/klog/v2"
)

// IsRetryableErrorFunc allows the client to provide its own function
// that determines whether the specified err from the server is retryable.
//
// request: the original request sent to the server
// err: the server sent this error to us
//
// The function returns true if the error is retryable and the request
// can be retried, otherwise it returns false.
// We have four mode of communications - 'Stream', 'Watch', 'Do' and 'DoRaw', this
// function allows us to customize the retryability aspect of each.
type IsRetryableErrorFunc func(request *http.Request, err error) bool

func (r IsRetryableErrorFunc) IsErrorRetryable(request *http.Request, err error) bool {
	return r(request, err)
}

var neverRetryError = IsRetryableErrorFunc(func(_ *http.Request, _ error) bool {
	return false
})

// WithRetry allows the client to retry a request up to a certain number of times
// Note that WithRetry is not safe for concurrent use by multiple
// goroutines without additional locking or coordination.
type WithRetry interface {
	// SetMaxRetries makes the request use the specified integer as a ceiling
	// for retries upon receiving a 429 status code  and the "Retry-After" header
	// in the response.
	// A zero maxRetries should prevent from doing any retry and return immediately.
	SetMaxRetries(maxRetries int)

	// IsNextRetry advances the retry counter appropriately
	// and returns true if the request should be retried,
	// otherwise it returns false, if:
	//  - we have already reached the maximum retry threshold.
	//  - the error does not fall into the retryable category.
	//  - the server has not sent us a 429, or 5xx status code and the
	//    'Retry-After' response header is not set with a value.
	//  - we need to seek to the beginning of the request body before we
	//    initiate the next retry, the function should log an error and
	//    return false if it fails to do so.
	//
	// restReq: the associated rest.Request
	// httpReq: the HTTP Request sent to the server
	// resp: the response sent from the server, it is set if err is nil
	// err: the server sent this error to us, if err is set then resp is nil.
	// f: a IsRetryableErrorFunc function provided by the client that determines
	//    if the err sent by the server is retryable.
	IsNextRetry(ctx context.Context, restReq *Request, httpReq *http.Request, resp *http.Response, err error, f IsRetryableErrorFunc) bool

	// Before should be invoked prior to each attempt, including
	// the first one. if an error is returned, the request
	// should be aborted immediately.
	Before(ctx context.Context, r *Request) error

	// After should be invoked immediately after an attempt is made.
	After(ctx context.Context, r *Request, resp *http.Response, err error)
}

// RetryAfter holds information associated with the next retry.
type RetryAfter struct {
	// Wait is the duration the server has asked us to wait before
	// the next retry is initiated.
	// This is the value of the 'Retry-After' response header in seconds.
	Wait time.Duration

	// Attempt is the Nth attempt after which we have received a retryable
	// error or a 'Retry-After' response header from the server.
	Attempt int

	// Reason describes why we are retrying the request
	Reason string
}

type withRetry struct {
	maxRetries int
	attempts   int

	// retry after parameters that pertain to the attempt that is to
	// be made soon, so as to enable 'Before' and 'After' to refer
	// to the retry parameters.
	//  - for the first attempt, it will always be nil
	//  - for consecutive attempts, it is non nil and holds the
	//    retry after parameters for the next attempt to be made.
	retryAfter *RetryAfter
}

func (r *withRetry) SetMaxRetries(maxRetries int) {
	if maxRetries < 0 {
		maxRetries = 0
	}
	r.maxRetries = maxRetries
}

func (r *withRetry) IsNextRetry(ctx context.Context, restReq *Request, httpReq *http.Request, resp *http.Response, err error, f IsRetryableErrorFunc) bool {
	if httpReq == nil || (resp == nil && err == nil) {
		// bad input, we do nothing.
		return false
	}

	r.attempts++
	r.retryAfter = &RetryAfter{Attempt: r.attempts}
	if r.attempts > r.maxRetries {
		return false
	}

	// if the server returned an error, it takes precedence over the http response.
	var errIsRetryable bool
	if f != nil && err != nil && f.IsErrorRetryable(httpReq, err) {
		errIsRetryable = true
		// we have a retryable error, for which we will create an
		// artificial "Retry-After" response.
		resp = retryAfterResponse()
	}
	if err != nil && !errIsRetryable {
		return false
	}

	// if we are here, we have either a or b:
	//  a: we have a retryable error, for which we already
	//     have an artificial "Retry-After" response.
	//  b: we have a response from the server for which we
	//     need to check if it is retryable
	seconds, wait := checkWait(resp)
	if !wait {
		return false
	}

	r.retryAfter.Wait = time.Duration(seconds) * time.Second
	r.retryAfter.Reason = getRetryReason(r.attempts, seconds, resp, err)

	if err := r.prepareForNextRetry(ctx, restReq); err != nil {
		klog.V(4).Infof("Could not retry request - %v", err)
		return false
	}

	return true
}

// prepareForNextRetry is responsible for carrying out operations that need
// to be completed before the next retry is initiated:
// - if the request context is already canceled there is no need to
//   retry, the function will return ctx.Err().
// - we need to seek to the beginning of the request body before we
//   initiate the next retry, the function should return an error if
//   it fails to do so.
func (r *withRetry) prepareForNextRetry(ctx context.Context, request *Request) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Ensure the response body is fully read and closed before
	// we reconnect, so that we reuse the same TCP connection.
	if seeker, ok := request.body.(io.Seeker); ok && request.body != nil {
		if _, err := seeker.Seek(0, 0); err != nil {
			return fmt.Errorf("can't Seek() back to beginning of body for %T", request)
		}
	}

	klog.V(4).Infof("Got a Retry-After %s response for attempt %d to %v", r.retryAfter.Wait, r.retryAfter.Attempt, request.URL().String())
	return nil
}

func (r *withRetry) Before(ctx context.Context, request *Request) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	url := request.URL()

	// r.retryAfter represents the retry after parameters calculated
	// from the (response, err) tuple from the last attempt, so 'Before'
	// can apply these retry after parameters prior to the next attempt.
	// 'r.retryAfter == nil' indicates that this is the very first attempt.
	if r.retryAfter == nil {
		// we do a backoff sleep before the first attempt is made,
		// (preserving current behavior).
		request.backoff.Sleep(request.backoff.CalculateBackoff(url))
		return nil
	}

	// if we are here, we have made attempt(s) al least once before.
	if request.backoff != nil {
		// TODO(tkashem) with default set to use exponential backoff
		//  we can merge these two sleeps:
		//  BackOffManager.Sleep(max(backoffManager.CalculateBackoff(), retryAfter))
		//  see https://github.com/kubernetes/kubernetes/issues/108302
		request.backoff.Sleep(r.retryAfter.Wait)
		request.backoff.Sleep(request.backoff.CalculateBackoff(url))
	}

	// We are retrying the request that we already send to
	// apiserver at least once before. This request should
	// also be throttled with the client-internal rate limiter.
	if err := request.tryThrottleWithInfo(ctx, r.retryAfter.Reason); err != nil {
		return err
	}

	return nil
}

func (r *withRetry) After(ctx context.Context, request *Request, resp *http.Response, err error) {
	// 'After' is invoked immediately after an attempt is made, let's label
	// the attempt we have just made as attempt 'N'.
	// the current value of r.retryAfter represents the retry after
	// parameters calculated from the (response, err) tuple from
	// attempt N-1, so r.retryAfter is outdated and should not be
	// referred to here.
	r.retryAfter = nil

	if request.c.base != nil {
		if err != nil {
			request.backoff.UpdateBackoff(request.URL(), err, 0)
		} else {
			request.backoff.UpdateBackoff(request.URL(), err, resp.StatusCode)
		}
	}
}

// checkWait returns true along with a number of seconds if
// the server instructed us to wait before retrying.
func checkWait(resp *http.Response) (int, bool) {
	switch r := resp.StatusCode; {
	// any 500 error code and 429 can trigger a wait
	case r == http.StatusTooManyRequests, r >= 500:
	default:
		return 0, false
	}
	i, ok := retryAfterSeconds(resp)
	return i, ok
}

func getRetryReason(retries, seconds int, resp *http.Response, err error) string {
	// priority and fairness sets the UID of the FlowSchema
	// associated with a request in the following response Header.
	const responseHeaderMatchedFlowSchemaUID = "X-Kubernetes-PF-FlowSchema-UID"

	message := fmt.Sprintf("retries: %d, retry-after: %ds", retries, seconds)

	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		// it is server-side throttling from priority and fairness
		flowSchemaUID := resp.Header.Get(responseHeaderMatchedFlowSchemaUID)
		return fmt.Sprintf("%s - retry-reason: due to server-side throttling, FlowSchema UID: %q", message, flowSchemaUID)
	case err != nil:
		// it's a retryable error
		return fmt.Sprintf("%s - retry-reason: due to retryable error, error: %v", message, err)
	default:
		return fmt.Sprintf("%s - retry-reason: %d", message, resp.StatusCode)
	}
}

func readAndCloseResponseBody(resp *http.Response) {
	if resp == nil {
		return
	}

	// Ensure the response body is fully read and closed
	// before we reconnect, so that we reuse the same TCP
	// connection.
	const maxBodySlurpSize = 2 << 10
	defer resp.Body.Close()

	if resp.ContentLength <= maxBodySlurpSize {
		io.Copy(ioutil.Discard, &io.LimitedReader{R: resp.Body, N: maxBodySlurpSize})
	}
}

func retryAfterResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     http.Header{"Retry-After": []string{"1"}},
	}
}
