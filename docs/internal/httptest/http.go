// Copyright (c) 2022 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package httptest

import (
	"net/http"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/docs/internal/iotest"
	"go.uber.org/fx/docs/internal/test"
)

// PostSuccess makes an HTTP POST request to the given URL,
// and returns the response body.
//
// PostSuccess uses a text/plain content type,
// and expects a 200 status code.
func PostSuccess(t test.T, url, body string) string {
	t.Helper()

	res, err := http.Post(url, "text/plain", strings.NewReader(body))
	require.NoError(t, err, "http post %q", url)
	defer func() {
		assert.NoError(t, res.Body.Close(), "close response body")
	}()
	assert.Equal(t, http.StatusOK, res.StatusCode, "status code did not match")

	return iotest.ReadAll(t, res.Body)
}
