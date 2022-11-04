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

package exectest

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx/docs/internal/test"
)

// Command builds an exec.Cmd that will run the given function as an external
// executable.
//
// This operates by re-running the test executable to run only the current
// test, and hijacking that test execution to run the main function.
//
//	cmd := Command(t, func() { fmt.Println("hello") })
//	got, err := cmd.Output()
//	...
//	fmt.Println(string(got) == "hello\n") // true
func Command(t test.T, main func()) *exec.Cmd {
	t.Helper()

	// This messes up the hijacking sometimes.
	// Keep it simple -- only top level tests can do this.
	require.NotContains(t, t.Name(), "/",
		"exectest.Command cannot be used with subtests")

	if filepath.Base(os.Args[0]) == t.Name() {
		// We can't get coverage for this block
		// because if the condition is true,
		// we're inside the subprocess.
		main()
		os.Exit(0)
	}

	exe, err := os.Executable()
	require.NoError(t, err, "determine executable")

	cmd := exec.Command(exe, "-test.run", "^"+t.Name()+"$")
	// Args[0] is the value of os.Args[0] for the new executable.
	// os.Args[0] is allowed to be different from the command.
	cmd.Args[0] = t.Name()
	return cmd
}
