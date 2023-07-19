package blast

import (
	"fmt"
	"sync"
	"time"

	humanize "github.com/dustin/go-humanize"
)

type Result struct {
	Path  string
	Hash  string
	Error error
	Size  uint64
}

type Summarizer struct {
	StartTime    time.Time
	EndTime      time.Time
	TotalFiles   uint64
	TotalSuccess uint64
	TotalBytes   uint64
	TotalFailed  uint64

	wg    *sync.WaitGroup
	rchan chan *Result
}

func NewSummarizer(wg *sync.WaitGroup) (*Summarizer, chan<- *Result) {
	rch := make(chan *Result, 1)
	s := &Summarizer{
		wg:    wg,
		rchan: rch,
	}

	return s, rch
}

func (s *Summarizer) Run() {

	// kick of a goroutine to wait on the waitgroup and close a
	//  channel when it is over
	done := make(chan bool)
	go func() {
		s.wg.Wait()
		done <- true
	}()

	// start running
	s.StartTime = time.Now()
	for {
		var ok bool
		var result *Result

		select {
		case <-done:
			// close the input channel to signal that we are over.. so that
			//   any buffered messages are handled before quitting
			close(s.rchan)

		case result, ok = <-s.rchan:
		}

		if ok == false {
			break
		}

		s.TotalFiles += 1
		if result.Error == nil {
			fmt.Printf("-> %s (%s)\n", result.Path, humanize.Bytes(result.Size))
			s.TotalSuccess += 1
			s.TotalBytes += result.Size
		} else {
			fmt.Printf("-> failed: %s\n", result.Path)
			fmt.Printf("     %s\n", result.Error)
			s.TotalFailed += 1
		}
	}
	s.EndTime = time.Now()
}

func (s *Summarizer) Report() {

	duration := s.EndTime.UnixMilli() - s.StartTime.UnixMilli()
	if duration == 0 {
		duration = 1
	}
	rate := uint64(float64(s.TotalBytes) / float64(duration) * 1000.0)

	fmt.Println("upoad summary")
	fmt.Printf("      runtime: %s sec\n", humanize.Comma(duration/1000.0))
	fmt.Printf("  total files: %d\n", s.TotalFiles)
	fmt.Printf("      success: %d\n", s.TotalFiles)
	fmt.Printf("     failures: %d\n", s.TotalFailed)
	fmt.Printf("        bytes: %s\n", humanize.Bytes(s.TotalBytes))
	fmt.Printf("         rate: %s/s\n", humanize.Bytes(rate))
}
