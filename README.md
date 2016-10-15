# S3 wiki

A simple Wiki system written in golang.

* Amazon S3 (and its compatibles) back-end
* GitHub Flavored Markdown
* Easy to share a page to the public
* Not searchable :p

## Build and run

Get dependencies

~~~
go get -v
~~~

Set your bucket name and credentials to environment variable.

~~~
AWS_BUCKET_NAME=<bucket name>
AWS_BUCKET_REGION=<region name>
AWS_ACCESS_KEY_ID=<access key>
AWS_SECRET_ACCESS_KEY=<secret access keys>
TWITTER_KEY=<twitter key>
TWITTER_SECRET=<twitter secret>
URL=<external URL for callback http://localhost:8080>
UUID=<arbitrary UUID for your app>
~~~

Build

~~~
go build
~~~

Run and access http://localhost:8080/
