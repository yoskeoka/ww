package testutil

import (
	"bytes"
	"testing"
)

func TestReadCombinedOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "multiplexed header",
			input: dockerFrame(1, []byte("hello")),
			want:  "hello",
		},
		{
			name:  "plain ascii",
			input: []byte("hello world\n"),
			want:  "hello world\n",
		},
		{
			name:  "short input",
			input: []byte("abc"),
			want:  "abc",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := readCombinedOutput(bytes.NewReader(tc.input))
			if err != nil {
				t.Fatalf("readCombinedOutput: %v", err)
			}
			if got != tc.want {
				t.Fatalf("readCombinedOutput() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsDockerMultiplexedHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		h    []byte
		want bool
	}{
		{
			name: "valid stdout frame",
			h:    []byte{1, 0, 0, 0, 0, 0, 0, 5},
			want: true,
		},
		{
			name: "plain printable ascii",
			h:    []byte("hello!!!"),
			want: false,
		},
		{
			name: "short header",
			h:    []byte{1, 0, 0, 0, 0, 0, 0},
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isDockerMultiplexedHeader(tc.h); got != tc.want {
				t.Fatalf("isDockerMultiplexedHeader(%v) = %v, want %v", tc.h, got, tc.want)
			}
		})
	}
}

func dockerFrame(stream byte, payload []byte) []byte {
	frame := make([]byte, 8+len(payload))
	frame[0] = stream
	frame[4] = byte(len(payload) >> 24)
	frame[5] = byte(len(payload) >> 16)
	frame[6] = byte(len(payload) >> 8)
	frame[7] = byte(len(payload))
	copy(frame[8:], payload)
	return frame
}
