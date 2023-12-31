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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/studio1767/s3blast/internal/blast"
)

func main() {
	// process the command line
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <files> <s3url>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	profile := flag.String("p", "default", "aws s3 credentials profile")
	endpoint_url := flag.String("e", "", "endpoint url")
	rredundancy := flag.Bool("r", false, "use reduced redundancy storage class")
	ignoredot := flag.Bool("d", false, "ignore dot-files and dot-directories")
	nworkers := flag.Int("n", 2, "the number of upload workers")
	maxfiles := flag.Int("c", -1, "max number of files to upload")
	verbose := flag.Bool("v", false, "verbose progress reporting")

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

	// create the confit
	cfg, err := s3Config(*profile, *endpoint_url)
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
	srizer, woChan := blast.NewSummarizer(&wg, *verbose)

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

	os.Exit(srizer.ExitStatus())
}

func s3Config(profile, endpoint_url string) (aws.Config, error) {
	var functions []func(*config.LoadOptions) error

	if endpoint_url != "" {
		resolver := aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				ep := aws.Endpoint{
					URL: endpoint_url,
				}
				return ep, nil
			},
		)
		functions = append(functions, config.WithEndpointResolverWithOptions(resolver))
	}
	functions = append(functions, config.WithSharedConfigProfile(profile))

	cfg, err := config.LoadDefaultConfig(context.Background(), functions...)
	return cfg, err
}
