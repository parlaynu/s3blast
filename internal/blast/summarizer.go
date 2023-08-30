package blast

import (
	"fmt"
	"sync"
	"time"

	humanize "github.com/dustin/go-humanize"
)

type action int

const (
	UPLOADED action = iota
	MATCHED
	SKIPPED
	OVERWROTE
	FAILED
)

var (
	actions = [...]string{
		"UPLOADED",
		"MATCHED",
		"SKIPPED",
		"OVERWROTE",
		"FAILED",
	}
)

type Result struct {
	Action action
	Path   string
	Hash   string
	Size   uint64
	Error  error
}

type Summarizer struct {
	StartTime      time.Time
	EndTime        time.Time
	TotalFiles     uint64
	TotalUploaded  uint64
	TotalMatched   uint64
	TotalSkipped   uint64
	TotalOverwrote uint64
	TotalFailed    uint64
	TotalBytes     uint64

	verbose bool

	wg    *sync.WaitGroup
	rchan chan *Result
}

func NewSummarizer(wg *sync.WaitGroup, verbose bool) (*Summarizer, chan<- *Result) {
	rch := make(chan *Result, 1)
	s := &Summarizer{
		verbose: verbose,
		wg:      wg,
		rchan:   rch,
	}

	return s, rch
}

func (s *Summarizer) Run() {

	// work out some formatting
	actionLen := 0
	for _, action := range actions {
		if len(action) > actionLen {
			actionLen = len(action)
		}
	}
	format := fmt.Sprintf("-> %%-%ds %%s (%%s)\n", actionLen)

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
		switch result.Action {
		case UPLOADED:
			s.TotalUploaded += 1
			s.TotalBytes += result.Size
		case MATCHED:
			s.TotalMatched += 1
		case SKIPPED:
			s.TotalSkipped += 1
		case OVERWROTE:
			s.TotalOverwrote += 1
		case FAILED:
			s.TotalFailed += 1
		}

		if s.verbose || result.Action != MATCHED {
			fmt.Printf(format, actions[result.Action], result.Path, humanize.Bytes(result.Size))
			if result.Error != nil {
				fmt.Printf("     %s\n", result.Error)
			}
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
	fmt.Printf("     uploaded: %d\n", s.TotalUploaded)
	fmt.Printf("        bytes: %s\n", humanize.Bytes(s.TotalBytes))
	fmt.Printf("      matched: %d\n", s.TotalMatched)
	fmt.Printf("      skipped: %d\n", s.TotalSkipped)
	fmt.Printf("    overwrote: %d\n", s.TotalOverwrote)
	fmt.Printf("     failures: %d\n", s.TotalFailed)
	fmt.Printf("         rate: %s/s\n", humanize.Bytes(rate))
}
