# Image Server

[![Build Status](https://travis-ci.org/image-server/image-server.svg)](https://travis-ci.org/image-server/image-server)

## Server

### Posting New Images

An image needs to be uploaded to a namespace.

Namespaces allow to group image types. For example avatars will require different image sizes than product images.

Uploading an image requires a source, which is the URL of the original image.

```shell
curl -X POST http://localhost:7000/p?source=http://example.com/image.jpg
```

A binary file might be uploaded instead of providing an URL. The contents of the image need to be included in the body of the request.

```shell
> curl --data-binary "@./test/images/wine.jpg" -X POST http://localhost:7000/p
{
  "hash": "6e0072682e66287b662827da75b244a3",
  "height": 600,
  "width": 800,
  "content_type": "image/jpeg"
}
```

It is possible to process images when uploading an image by providing the desired image dimensions in the `outputs` parameter.

```shell
> curl --data-binary "@./test/images/wine.jpg" -X POST http://localhost:7000/p?outputs=x300.jpg,x300.webp
{
  "hash": "6e0072682e66287b662827da75b244a3",
  "height": 496,
  "width": 574,
  "content_type": "image/jpeg"
}
```

An upload request will block till all images have been created (various sizes) _and_ uploaded to S3

### Image Information

The request returns the _"Image Information"_ after an image is uploaded. The response includes properties of the image, and the image hash to be used to retrieve it in the future.

Image properties can be retrieved by visiting the info page. The response is the same as the one returned when creating the image. Please note url has partitioned the image hash `/123/456/789/<REST_OF_HASH>/`

```shell
> curl http://localhost:7000/p/6e0/072/682/e66287b662827da75b244a3/info.json
{
  "hash": "6e0072682e66287b662827da75b244a3",
  "height": 496,
  "width": 574,
  "content_type": "image/jpeg"
}
```

### Image processing

Images can be processed on demand. This will re-size and also upload the image to the configured data store!

**Dimensions**

    By Width
    GET http://localhost:7000/p/6e0/072/682/e66287b662827da75b244a3/w200.jpg

![Image](test/images/wine/w200.jpg?raw=true)

    Square
    GET http://localhost:7000/p/6e0/072/682/e66287b662827da75b244a3/x200.jpg

![Image](test/images/wine/x200.jpg?raw=true)

    Rectangle (width x height)
    GET http://localhost:7000/p/6e0/072/682/e66287b662827da75b244a3/300x200.jpg

![Image](test/images/wine/300x200.jpg?raw=true)

**Quality**

The default compression of the image can be modified by appending `-q` and the desired quality `1-100`.

    Square with quality 50
    GET http://localhost:7000/p/6e0/072/682/e66287b662827da75b244a3/x200-q30.jpg

![Image](test/images/wine/x200-q30.jpg?raw=true)

### Cloud Storage

Images can be uploaded to either Amazon S3

To store images in S3 the following flags need to be set

```shell
--aws_access_key_id $AWS_ACCESS_KEY_ID --aws_secret_key $AWS_SECRET_KEY --aws_bucket $AWS_BUCKET --aws_region us-west-1
```

### Error Handling

Few errors will cause the server to return error pages

- Source image is not found: NotFound (404)

## CLI

Images can be processed with the command line.

## Development

Set up the environment:

```bash
mkdir -p $GOPATH/src/github.com/image-server/
git clone https://github.com/image-server/image-server $GOPATH/src/github.com/image-server/image-server
ln -s $GOPATH/src/github.com/image-server/image-server ~/workspace/image-server
cd ~/workspace/image-server
```

Install dependencies:

Go needs to be installed with cross compilation. Imagemagick will require giflib and webp support.

On Mac

```bash
brew install --force go --with-cc-all
brew install --force giflib
brew install --force imagemagick --with-webp
make deps
```

Set up editor:

- Atom.io package [go-plus](https://github.com/joefitzgerald/go-plus)

Compile the app:

To build the executables under `./bin`

```bash
make build
```

## Development Usage

There are few `make` helpers that start the development server. They all translate environment variables into flags.

### S3

Required ENV variables: `IMG_OUTPUTS`, `AWS_BUCKET`, `IMG_REMOTE_BASE_PATH`, `IMG_REMOTE_BASE_URL`

```
make dev-server-s3
```

### No uploader, only store images locally

Required ENV variables: `IMG_OUTPUTS`

```bash
make dev-server
```

## Tests

S3 tests make real HTTP calls when you have S3 ENV variables set.

```
make tests
```

## Configuration

All configuration is passed by flags

go run main.go --help

## Deploy

Make sure you increase the version number in core/version.go

```bash
make release
```

## Statsd

To enable the statsd events, use the flag `enable_statsd`:

```
bin/images --enable_statsd
```

### Statsd Events

A local cache was not found and the image was processed. This also tracks count of images sent to remote store.

```
stats.image_server.image_request
```

In addition, the format is tracked (jpg, gif, webp)

```
stats.image_server.image_request.jpg
```

Request failed to return an image

```
stats.image_server.image_request_fail
```

Every download from original source, and a 404 was returned

```
stats.image_server.original_downloaded
```

The original image is not available, and a 404 was returned

```
stats.image_server.original_unavailable
```

## Prometheus metrics

Prometheus metrics are available on the admin port at `/metrics`

## Profiling

The server allows to be profiled when started with the profile flag

```
bin/images --profile server
```

The profiling information is available on `localhost:6060`

It is important to run the profiler

You will need the profiled data from the server, and analize it with the same executable file used on the server.

You will need to download the profiled data from the server.

```
ssh example.com "curl http://localhost:6060/debug/pprof/heap" > images.pprof
```

Use `go tool pprof` to analize the profile. Remember to use the same executable file as the one on production.

```
go tool pprof --inuse_objects bin/solaris/images images.pprof
```

## Benchmarks

Make sure your computer can handle enough simultaneous connections. MacOS X by default allows 128. Need a lot more!

```shell
$ sudo sysctl -w kern.ipc.somaxconn=2048
```

Also need to increase the limit of maximum open files

To find out the limits on your computer:

```shell
launchctl limit
```

Increase the limits!

```shell
launchctl limit maxfiles 400000 1000000
```

to increase them [permanently](https://coderwall.com/p/lfjoaq)
