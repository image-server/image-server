FROM golang:1.9-alpine3.6

MAINTAINER Thomas Barton

WORKDIR ${GOPATH}/src/github.com/image-server/image-server

COPY . .

ARG SHORT_COMMIT_HASH

RUN go build -ldflags="-X github.com/image-server/image-server/core.BuildStamp=`date -u '+%Y-%m-%d_%I:%M:%S%p_%z'` -X github.com/image-server/image-server/core.GitHash=${SHORT_COMMIT_HASH}"

FROM alpine:3.6

RUN apk add --no-cache imagemagick
RUN apk add --no-cache ca-certificates

WORKDIR /opt/image-server

RUN mkdir -p public
RUN chmod 775 -R public

COPY start.sh .

RUN mkdir -p bin
COPY --from=0 /go/src/github.com/image-server/image-server/image-server bin/image-server
RUN chmod 775 -R bin/image-server

EXPOSE 7000
EXPOSE 7002

CMD ["./start.sh"]