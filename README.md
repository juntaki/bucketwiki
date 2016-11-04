# Bucket Wiki

A simple Wiki system written in golang.

* Amazon S3 back-end, No need to setup/backup database.
* Easy to share a page to the public



## Build and run

Set your bucket name and credentials to environment variable.

~~~
AWS_BUCKET_NAME=<bucket name>
AWS_BUCKET_REGION=<region name>
AWS_ACCESS_KEY_ID=<access key>
AWS_SECRET_ACCESS_KEY=<secret access keys>
TWITTER_KEY=<twitter key>
TWITTER_SECRET=<twitter secret>
URL=<external URL for callback like http://localhost:8080>
WIKI_ID=<arbitrary string for your wiki>
SESSION_SECRET=<Session secret>
~~~

Get dependencies and build

~~~
go get -v
go build
~~~

Run and access http://localhost:8080/

## Run on Docker

~~~
docker run -d -p 8080:8080\
    -e AWS_BUCKET_NAME="<bucket name>" \
    -e AWS_BUCKET_REGION="<region name>" \
    -e AWS_ACCESS_KEY_ID="<access key>" \
    -e AWS_SECRET_ACCESS_KEY="<secret access keys>" \
    -e TWITTER_KEY="<twitter key>" \
    -e TWITTER_SECRET="<twitter secret>" \
    -e URL="<external URL for callback like http://localhost:8080>" \
    -e WIKI_ID="<arbitrary string for your wiki>" \
    -e SESSION_SECRET="<Session secret>" \
    juntaki/BucketWiki
~~~

## License

MIT

## Author

juntaki 
