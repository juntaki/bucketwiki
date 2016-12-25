# Bucket Wiki

A simple Wiki system written in golang.

[![Build Status](https://travis-ci.org/juntaki/bucketwiki.svg?branch=master)](https://travis-ci.org/juntaki/bucketwiki)
[![Coverage Status](https://coveralls.io/repos/github/juntaki/bucketwiki/badge.svg?branch=master)](https://coveralls.io/github/juntaki/bucketwiki?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/juntaki/bucketwiki)](https://goreportcard.com/report/github.com/juntaki/bucketwiki)

![screenshot](https://github.com/juntaki/bucketwiki/blob/master/screenshot.gif?raw=true)

* Amazon S3 back-end, No need to setup/backup database.
* Easy to share a page to the public

## Deploy to Heroku

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

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
WIKI_SECRET=<arbitrary string for your wiki>
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
    -e WIKI_SECRET="<arbitrary string for your wiki>" \
    juntaki/bucketwiki
~~~
