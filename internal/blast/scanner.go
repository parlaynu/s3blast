package blast

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func NewScanner(root string, ignoredot bool, maxfiles int) (<-chan string, error) {
	// quick sanity check
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	ch := make(chan string, 10)

	if info.Mode().IsRegular() {
		// a single file... put it on the channel immediately
		// ... and close it to signal all done
		ch <- root
		close(ch)

	} else if info.Mode().IsDir() {
		go func() {
			defer close(ch)
			scan(ch, root, ignoredot, maxfiles, 0)
		}()
	}

	return ch, nil
}

func scan(ch chan<- string, cdir string, ignoredot bool, maxfiles, curfiles int) int {
	entries, err := os.ReadDir(cdir)
	if err != nil {
		log.Printf("failed to read dir %s with %s", cdir, err)
		return curfiles
	}

	for _, entry := range entries {
		if maxfiles > 0 && curfiles >= maxfiles {
			break
		}
		if ignoredot && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		fpath := filepath.Join(cdir, entry.Name())

		if entry.Type().IsRegular() {
			ch <- fpath
			curfiles += 1
		} else if entry.Type().IsDir() {
			curfiles = scan(ch, fpath, ignoredot, maxfiles, curfiles)
		}
	}

	return curfiles
}
