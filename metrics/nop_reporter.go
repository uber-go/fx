// Copyright (c) 2017 Uber Technologies, Inc.
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

package metrics

import (
	"time"

	"github.com/uber-go/tally"
)

var (
	capabilitiesReporting = &capabilities{
		reporting: true,
		tagging:   true,
	}
)

type capabilities struct {
	reporting bool
	tagging   bool
}

func (c *capabilities) Reporting() bool {
	return c.reporting
}

func (c *capabilities) Tagging() bool {
	return c.tagging
}

// NopCachedStatsReporter is an implementation of tally.CachedStatsReporter that simply does nothing
// and should be used for testing purposes only.
// TODO:(anup) This should exist in tally. https://github.com/uber-go/tally/issues/23
// Remove and replace metrics.NopCachedStatsReporter with tally.NopCachedStatsReporter once issue is resolved
var NopCachedStatsReporter tally.CachedStatsReporter = nopCachedStatsReporter{}

type nopCachedStatsReporter struct{}

func (nopCachedStatsReporter) AllocateCounter(name string, tags map[string]string) tally.CachedCount {
	return NopCachedCount
}

func (nopCachedStatsReporter) AllocateGauge(name string, tags map[string]string) tally.CachedGauge {
	return NopCachedGauge
}

func (nopCachedStatsReporter) AllocateHistogram(name string, tags map[string]string, buckets tally.Buckets) tally.CachedHistogram {
	return NopCachedHistogram
}

func (nopCachedStatsReporter) AllocateTimer(name string, tags map[string]string) tally.CachedTimer {
	return NopCachedTimer
}

func (r nopCachedStatsReporter) Capabilities() tally.Capabilities {
	return capabilitiesReporting
}

func (r nopCachedStatsReporter) Flush() {}

// NopCachedCount is an implementation of tally.CachedCount and same as other Nop types don't do anything and
// should be used for testing purposes only.
var NopCachedCount tally.CachedCount = nopCachedCount{}

type nopCachedCount struct {
}

func (nopCachedCount) ReportCount(value int64) {
}

// NopCachedGauge is an implementation of tally.CachedGauge and same as other Nop types don't do anything and
// should be used for testing purposes only.
var NopCachedGauge tally.CachedGauge = nopCachedGauge{}

type nopCachedGauge struct{}

func (nopCachedGauge) ReportGauge(value float64) {}

// NopCachedTimer is an implementation of tally.CachedTimer
var NopCachedTimer tally.CachedTimer = nopCachedTimer{}

type nopCachedTimer struct{}

func (nopCachedTimer) ReportTimer(interval time.Duration) {}

// NopCachedHistogram is an implementation of tally.CachedHistogram and same as other Nop types don't do anything and
// should be used for testing purposes only.
var NopCachedHistogram tally.CachedHistogram = nopCachedHistogram{}

type nopCachedHistogram struct{}

func (nopCachedHistogram) ValueBucket(
	bucketLowerBound float64,
	bucketUpperBound float64,
) tally.CachedHistogramBucket {
	return nopCachedHistogramBucket{}
}

func (nopCachedHistogram) DurationBucket(
	bucketLowerBound time.Duration,
	bucketUpperBound time.Duration,
) tally.CachedHistogramBucket {
	return nopCachedHistogramBucket{}
}

// NopCachedHistogramBucket is an implementation of tally.CachedHistogramBucket and same as other Nop types
// don't do anything and should be used for testing purposes only.
var NopCachedHistogramBucket tally.CachedHistogramBucket = nopCachedHistogramBucket{}

type nopCachedHistogramBucket struct{}

func (nopCachedHistogramBucket) ReportSamples(value int64) {}
