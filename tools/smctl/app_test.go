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

package smctl

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

const getNamespaceStateStubMsg = "Get namespace state is not implemented yet!"

func TestBuildCommand_metadata(t *testing.T) {
	t.Parallel()
	cmd := BuildCommand()
	if cmd.Name != "smctl" {
		t.Fatalf("Name: got %q want smctl", cmd.Name)
	}
	if cmd.Usage == "" {
		t.Fatal("Usage should be set")
	}
	if cmd.Version == "" {
		t.Fatal("Version should be set")
	}
}

func TestBuildCommand_help(t *testing.T) {
	t.Parallel()
	cmd := BuildCommand()
	buf := new(bytes.Buffer)
	cmd.Writer = buf

	err := cmd.Run(context.Background(), []string{"smctl", "--help"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "smctl") {
		t.Fatalf("help output should mention smctl:\n%s", out)
	}
}

func TestGetNamespaceState_notImplemented(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "kebab", args: []string{"smctl", "get-namespace-state"}},
		{name: "camelAlias", args: []string{"smctl", "getNamespaceState"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := withStdoutCaptured(t, func() {
				cmd := BuildCommand()
				if err := cmd.Run(context.Background(), tt.args); err != nil {
					t.Fatalf("Run: %v", err)
				}
			})
			if !strings.Contains(out, getNamespaceStateStubMsg) {
				t.Fatalf("output: got %q want substring %q", out, getNamespaceStateStubMsg)
			}
		})
	}
}

// withStdoutCaptured runs fn while os.Stdout is a pipe; returns captured bytes as a string.
// Do not use t.Parallel in tests that call this (os.Stdout is process-global).
func withStdoutCaptured(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close pipe reader: %v", err)
	}
	return buf.String()
}
