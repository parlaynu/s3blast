# Upload Files to AWS S3 Bucket

A fairly simple application that uploads files from local storage to an S3 bucket. It scans a source
directory recursively looking for files, and uploads whatever is found.

It does two things to get uploads running in parallel:

* uploads multiple files at the same time (controlled by the number of goroutines)
* uses the s3 manager uploader so individual files upload in parallel if the file is large enough

Results from a very simple test, uploading 22MBytes in 150 files using a home internet connection:

| Goroutines | Rate         |
|------------|--------------|
| 1          | 653 kB/s     |
| 2          | 1.3 MB/s     |
| 3          | 1.7 MB/s     |
| 4          | 2.0 MB/s     |
| 5          | 2.1 MB/s     |
| 10         | 2.1 MB/s     |

How many goroutines will be optimal for you will vary depending on the file sizes and your internet connection
upload bandwidth. I noticed the upload speed decreasing with very large numbers of goroutines, so don't just set 
it at a ridiculous number and expect to get great results.

In one of my use cases, the uploads run 20% faster than using the aws cli. Your mileage will vary.

To run this tool, you will need:

* an AWS account
* an s3 bucket
* an AWS profile with permissions to upload to the bucket

If you can upload to the bucket using the AWS CLI, then this will work for you as well.

## Usage

The usage is:

    Usage: s3blast  [-p <profile>] [-n <numworkers>] [-d] <files> <bucket>/<key-prefix>
      -d	ignore dot-files and dot-directories
      -n int
        	the number of upload workers (default 2)
      -p string
        	aws s3 credentials profile (default "default")

The file root can be a directory in which case it is traversed recursively looking for files. Or, it can be
a single file.

An example:

    s3blast -p myprofile -n 4 ~/data mybucket/my/key/prefix/

This will recursively search the directory `~/data` looking for files to upload. 

In this example, the S3 key will be the prefix `my/key/prefix/` provided on the commandline joined with the relative path 
of the file under the `~/data` directory.

An equivalent upload command using the aws cli looks like this:

    aws --profile myprofile s3 cp --recursive ~/data s3://mybucket/my/key/prefix

