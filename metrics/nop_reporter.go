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
	capabilitiesReportingNoTagging = &capabilities{
		reporting: true,
		tagging:   false,
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

// NopCachedStatsReporter is an implementatin of tally.CachedStatsReporter than simply does nothing.
// TODO:(anup) This should exist in tally. https://github.com/uber-go/tally/issues/23
// Remove and replace metrics.NopCachedStatsReporter with tally.NopCachedStatsRepor once issue is resolved
var NopCachedStatsReporter tally.CachedStatsReporter = nopCachedStatsReporter{}

type nopCachedStatsReporter struct {
}

func (nopCachedStatsReporter) AllocateCounter(name string, tags map[string]string) tally.CachedCount {
	return NopCachedCount
}

func (nopCachedStatsReporter) AllocateGauge(name string, tags map[string]string) tally.CachedGauge {
	return NopCachedGauge
}

func (nopCachedStatsReporter) AllocateTimer(name string, tags map[string]string) tally.CachedTimer {
	return NopCachedTimer
}

func (r nopCachedStatsReporter) Capabilities() tally.Capabilities {
	return capabilitiesReportingNoTagging
}

func (r nopCachedStatsReporter) Flush() {
}

// NopCachedCount is an implementation of tally.CachedCount
var NopCachedCount tally.CachedCount = nopCachedCount{}

type nopCachedCount struct {
}

func (nopCachedCount) ReportCount(value int64) {
}

// NopCachedGauge is an implementation of tally.CachedGauge
var NopCachedGauge tally.CachedGauge = nopCachedGauge{}

type nopCachedGauge struct {
}

func (nopCachedGauge) ReportGauge(value float64) {
}

// NopCachedTimer is an implementation of tally.CachedTimer
var NopCachedTimer tally.CachedTimer = nopCachedTimer{}

type nopCachedTimer struct {
}

func (nopCachedTimer) ReportTimer(interval time.Duration) {
}
