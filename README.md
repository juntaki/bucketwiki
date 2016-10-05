# S3 wiki

A simple wiki written in golang.
Amazon S3 (and its compatibles) back-end.

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
~~~

Build

~~~
go build
~~~

Run and access http://localhost:8080/list
