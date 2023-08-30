package blast

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func NewWorker(wg *sync.WaitGroup, client *s3.Client, reducedRedundancy bool, bucket, prefix string, srcprefix string, iChan <-chan *Result, oChan chan<- *Result) error {
	// check the client and bucket work
	err := ping(client, bucket)
	if err != nil {
		return err
	}

	// get to work
	wg.Add(1)
	go func() {
		defer wg.Done()
		work(client, reducedRedundancy, bucket, prefix, srcprefix, iChan, oChan)
	}()

	return nil
}

func ping(client *s3.Client, bucket string) error {
	_, err := client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucket),
		Prefix:  aws.String(""),
		MaxKeys: 1,
	})
	return err
}

func work(client *s3.Client, reducedRedundancy bool, bucket, prefix string, srcprefix string, iChan <-chan *Result, oChan chan<- *Result) {

	ctx := context.Background()

	mdata := make(map[string]string)

	uploader := manager.NewUploader(client)

	for msg := range iChan {

		if msg.Error != nil {
			oChan <- msg
			continue
		}

		fpath := msg.Path

		// build the key
		rpath := strings.TrimPrefix(strings.TrimPrefix(fpath, srcprefix), "/")
		if rpath == "" {
			rpath = filepath.Base(fpath)
		}
		rpath = strings.Replace(rpath, string(filepath.Separator), "/", -1)

		key := path.Join(prefix, rpath)
		if prefix == "" {
			key = rpath
		}

		mdata["sha256"] = msg.Hash

		// check if the key exists and the hash matches
		exists, matches, err := existsAndMatches(ctx, client, bucket, key, mdata["sha256"])
		if err != nil {
			oChan <- &Result{
				Action: FAILED,
				Path:   fmt.Sprintf("%s/%s", bucket, key),
				Hash:   msg.Hash,
				Size:   0,
				Error:  err,
			}
			continue
		}
		if exists && matches {
			// no need to upload again...
			oChan <- &Result{
				Action: MATCHED,
				Path:   fmt.Sprintf("%s/%s", bucket, key),
				Hash:   msg.Hash,
				Size:   0,
				Error:  nil,
			}
			continue
		}

		// upload the file
		action := UPLOADED
		if exists {
			action = UPDATED
		}
		size, err := upload(ctx, uploader, reducedRedundancy, fpath, bucket, key, mdata)
		if err != nil {
			action = FAILED
		}
		oChan <- &Result{
			Action: action,
			Path:   fmt.Sprintf("%s/%s", bucket, key),
			Hash:   msg.Hash,
			Size:   size,
			Error:  err,
		}

	}
}

func existsAndMatches(ctx context.Context, client *s3.Client, bucket, key, hash string) (bool, bool, error) {

	heo, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		hhash, ok := heo.Metadata["sha256"]

		// the the hashes match...
		if ok && hhash == hash {
			return true, true, nil
		}

		// they don't match
		return true, false, nil
	}

	// check for a not found error...
	var responseError *awshttp.ResponseError
	if errors.As(err, &responseError) && responseError.ResponseError.HTTPStatusCode() == http.StatusNotFound {
		return false, false, nil
	}

	// it was some other sort of error, so return
	return false, false, err
}

func upload(
	ctx context.Context,
	uploader *manager.Uploader,
	reducedRedundancy bool,
	fpath, bucket, key string,
	mdata map[string]string) (uint64, error) {

	// open the file for reading
	freader, err := os.Open(fpath)
	if err != nil {
		return 0, err
	}
	defer freader.Close()

	sclass := types.StorageClassStandard
	if reducedRedundancy {
		sclass = types.StorageClassReducedRedundancy
	}

	// uplaod the file
	source := NewReadCounter(freader)
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		Body:         source,
		Metadata:     mdata,
		StorageClass: sclass,
	})
	if err != nil {
		return 0, err
	}

	return source.TotalBytes(), nil
}
