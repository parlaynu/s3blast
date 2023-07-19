package blast

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

func NewHasher(iChan <-chan string) (<-chan *Result, error) {

	oChan := make(chan *Result)

	go func() {
		defer close(oChan)
		hasher(iChan, oChan)
	}()

	return oChan, nil
}

func hasher(iChan <-chan string, oChan chan<- *Result) {

	for fpath := range iChan {
		hval, err := calcHash(fpath)
		oChan <- &Result{
			Path:  fpath,
			Hash:  hval,
			Error: err,
		}
	}
}

func calcHash(fpath string) (string, error) {
	// open for reading
	in, err := os.Open(fpath)
	if err != nil {
		return "", err
	}
	defer in.Close()

	// generate the hash
	h := sha256.New()
	_, err = io.Copy(h, in)
	if err != nil {
		return "", err
	}

	// return the value
	return hex.EncodeToString(h.Sum(nil)), nil
}
