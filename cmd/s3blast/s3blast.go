package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/studio1767/s3blast/internal/blast"
)

func main() {
	// process the command line
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s  [-p <profile>] [-n <numworkers>] [-d] <files> <bucket>/<key-prefix>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	profile := flag.String("p", "default", "aws s3 credentials profile")
	nworkers := flag.Int("n", 2, "the number of upload workers")
	ignoredot := flag.Bool("d", false, "ignore dot-files and dot-directories")
	rredundancy := flag.Bool("r", false, "use reduced redundancy storage class")

	flag.Parse()

	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "Error: incorrect arguments provided\n")
		flag.Usage()
		os.Exit(1)
	}

	srcroot := flag.Arg(0)

	s3path := flag.Arg(1)
	s3tokens := strings.Split(s3path, "/")
	bucket := s3tokens[0]
	keyprefix := ""
	if len(s3tokens) > 1 {
		keyprefix = path.Join(s3tokens[1:len(s3tokens)]...)
	}

	// create the client
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithSharedConfigProfile(*profile))
	if err != nil {
		log.Fatal(err)
	}
	client := s3.NewFromConfig(cfg)

	// create the file scanner
	wiChan, err := blast.NewScanner(srcroot, *ignoredot)
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
