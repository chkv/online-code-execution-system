package executor

import "bytes"

const (
	maxCapturedOutputBytes = 1 << 20 // 1 MiB per output stream.
	outputTruncationNotice = "\n[output truncated: 1 MiB limit]\n"
)

// limitedOutputBuffer retains bounded output and discards excess bytes.
// Read it only after the output-copy goroutine has finished.
type limitedOutputBuffer struct {
	buf       bytes.Buffer
	truncated bool
}

func (b *limitedOutputBuffer) Write(p []byte) (int, error) {
	received := len(p)
	remaining := maxCapturedOutputBytes - b.buf.Len()

	if received > remaining {
		p = p[:remaining]
		b.truncated = true
	}

	_, _ = b.buf.Write(p)

	// Report all bytes consumed so output copying continues normally.
	return received, nil
}

func (b *limitedOutputBuffer) String() string {
	captured := b.buf.String()
	if !b.truncated {
		return captured
	}

	// Include the notice within the output limit.
	keep := maxCapturedOutputBytes - len(outputTruncationNotice)
	return captured[:keep] + outputTruncationNotice
}
