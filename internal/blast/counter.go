package blast

import (
	"io"
)

type ReadCounter interface {
	Read(p []byte) (int, error)

	TotalReads() uint
	TotalBytes() uint64
}

func NewReadCounter(in io.Reader) ReadCounter {
	rc := readCounter{
		in:    in,
		reads: 0,
		bytes: 0,
	}
	return &rc
}

type readCounter struct {
	in    io.Reader
	reads uint
	bytes uint64
}

func (rc *readCounter) Read(p []byte) (int, error) {
	size, err := rc.in.Read(p)

	rc.reads += 1
	rc.bytes += uint64(size)

	return size, err
}

func (rc *readCounter) TotalReads() uint {
	return rc.reads
}

func (rc *readCounter) TotalBytes() uint64 {
	return rc.bytes
}
