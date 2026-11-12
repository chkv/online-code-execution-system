package executor

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/docker/docker/pkg/stdcopy"
)

func TestLimitedOutputBufferBoundaries(t *testing.T) {
	const limit = 1 << 20
	const notice = "\n[output truncated: 1 MiB limit]\n"

	tests := []struct {
		name string
		size int
	}{
		{"empty", 0},
		{"small", 128},
		{"below_limit", limit - 1},
		{"exactly_at_limit", limit},
		{"one_byte_over", limit + 1},
		{"large_write", 3 * limit},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buffer limitedOutputBuffer
			input := strings.Repeat("x", tc.size)

			n, err := io.WriteString(&buffer, input)
			if err != nil || n != len(input) {
				t.Fatalf("expected all %d bytes consumed, got %d, error: %v",
					len(input), n, err)
			}

			if retained := buffer.buf.Len(); retained > limit {
				t.Fatalf("buffer retained %d bytes, exceeding %d", retained, limit)
			}

			want := input
			if tc.size > limit {
				want = input[:limit-len(notice)] + notice
			}

			if got := buffer.String(); got != want {
				t.Fatalf("unexpected captured output: got %d bytes, expected %d",
					len(got), len(want))
			}
		})
	}
}

func TestLimitedOutputBufferDrainsDockerOutput(t *testing.T) {
	const limit = 1 << 20
	const notice = "\n[output truncated: 1 MiB limit]\n"

	var stream bytes.Buffer
	stdoutWriter := stdcopy.NewStdWriter(&stream, stdcopy.Stdout)
	stderrWriter := stdcopy.NewStdWriter(&stream, stdcopy.Stderr)

	const firstLine = "Beginning of stdout\n"
	if _, err := io.WriteString(stdoutWriter, firstLine); err != nil {
		t.Fatal(err)
	}

	// Multiple writes cross the limit and continue beyond it.
	chunk := bytes.Repeat([]byte("x"), 32*1024+1)
	for i := 0; i < 40; i++ {
		if _, err := stdoutWriter.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}

	const stderrText = "Stderr still arrives after excessive stdout\n"
	if _, err := io.WriteString(stderrWriter, stderrText); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr limitedOutputBuffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, &stream); err != nil {
		t.Fatalf("output copying failed: %v", err)
	}

	got := stdout.String()
	if len(got) != limit {
		t.Fatalf("expected stdout length %d, got %d", limit, len(got))
	}
	if !strings.HasPrefix(got, firstLine) {
		t.Fatal("the beginning of stdout was not preserved")
	}
	if !strings.HasSuffix(got, notice) {
		t.Fatal("stdout is missing the truncation notice")
	}
	if stderr.String() != stderrText {
		t.Fatal("stderr was not captured correctly after stdout reached its limit")
	}
	if stream.Len() != 0 {
		t.Fatalf("output stream still contains %d unread bytes", stream.Len())
	}
}
