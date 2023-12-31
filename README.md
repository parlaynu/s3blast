# AWS S3 and Cloudflare R2 Uploader

A fairly simple application that uploads files from local storage to an S3 or R2 bucket. It scans a source
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

Or:

* a Cloudflare account
* an r3 bucket
* permissions to upload to the bucket


If you can upload to the bucket using the AWS CLI, then this will work for you as well.

## Operation

This tool is designed so it can be run multiple times over the same source and bucket/key and
only new or modified files will be uploaded. 

Note: it never deletes files in the bucket. 

File contents are tracked using sha256 hashes. This hash of the file content is calculated during scanning
of the source and is also stored in the object in S3 as custom metadata when it is uploaded.

Files are only uploaded if they don't already exist in S3, or if the local and S3 content hashes
don't match.

Additionally, the exit status is zero if there were no failures, and non-zero if there were any failures. 
You could use this, for example, in a bash script to keep repeating the upload until it completes 
without failure.

## Usage

The usage is:

    Usage: s3blast [options] <files> <s3url>
      -c int
        	max number of files to upload (default -1)
      -d	ignore dot-files and dot-directories
      -n int
        	the number of upload workers (default 2)
      -p string
        	aws s3 credentials profile (default "default")
      -r	use reduced redundancy storage class
      -v	verbose progress reporting

The file root can be a directory in which case it is traversed recursively looking for files. Or, it can be
a single file.

An example:

    s3blast -p myprofile -n 4 ~/data s3://mybucket/my/key/prefix/

This will recursively search the directory `~/data` looking for files to upload. 

In this example, the S3 key will be the prefix `my/key/prefix/` provided on the commandline joined with the relative path 
of the file under the `~/data` directory.

An equivalent upload command using the aws cli looks like this:

    aws --profile myprofile s3 cp --recursive ~/data s3://mybucket/my/key/prefix

### R2 Profiles

A profile for using a R2 bucket is more involved than for an S3 bucket as you need to specify a `url_endpoint`.

The credentials file entry is the same as for S3, for example:

    [my_r2_bucket]
    aws_access_key_id = xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    aws_secret_access_key = yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy

The config entry is where you need to specify the `url_endpoint` and looks like this:

    [profile my_r2_bucket]
    output = json
    region = auto
    services = my_r2_bucket

    [services my_r2_bucket]
    s3 =
      endpoint_url = https://zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz.r2.cloudflarestorage.com

The credentials and account Id needed are all available from your Cloudflare console. 
Documentation is [here](https://developers.cloudflare.com/r2/).

