package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/studio1767/s3blast/internal/blast"
)

func main() {
	// process the command line
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s  [options] <files> <s3url>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	profile := flag.String("p", "default", "aws s3 credentials profile")
	rredundancy := flag.Bool("r", false, "use reduced redundancy storage class")
	ignoredot := flag.Bool("d", false, "ignore dot-files and dot-directories")
	nworkers := flag.Int("w", 2, "the number of upload workers")
	maxfiles := flag.Int("n", -1, "max number of files to upload")

	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "Error: incorrect arguments provided\n")
		flag.Usage()
		os.Exit(1)
	}

	srcroot := flag.Arg(0)
	s3url := flag.Arg(1)

	// process the s3url
	s3re := regexp.MustCompile("^s3://([^/]+)(/?)(.*)")

	s3tokens := s3re.FindStringSubmatch(s3url)
	if len(s3tokens) == 0 {
		fmt.Fprintf(os.Stderr, "Error: s3 url format must match: s3://bucket[/key/prefix/]\n")
		os.Exit(1)
	}
	bucket := s3tokens[1]
	keyprefix := s3tokens[3]

	// create the client
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithSharedConfigProfile(*profile))
	if err != nil {
		log.Fatal(err)
	}
	client := s3.NewFromConfig(cfg)

	// create the file scanner
	wiChan, err := blast.NewScanner(srcroot, *ignoredot, *maxfiles)
	if err != nil {
		log.Fatal(err)
	}

	// use a waitgroup to signal the summarizer that all workers are done
	var wg sync.WaitGroup
	srizer, woChan := blast.NewSummarizer(&wg)

	// get the workers running
	for i := 0; i < *nworkers; i++ {
		hoChan, err := blast.NewHasher(wiChan)
		if err != nil {
			log.Fatal(err)
		}
		err = blast.NewWorker(&wg, client, *rredundancy, bucket, keyprefix, srcroot, hoChan, woChan)
		if err != nil {
			log.Fatal(err)
		}
	}

	// run the summarizer and report
	srizer.Run()
	srizer.Report()
}
