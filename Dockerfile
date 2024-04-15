FROM golang:1.17.5-alpine3.15

LABEL maintainer="Thomas Barton"

WORKDIR ${GOPATH}/src/github.com/image-server/image-server

ENV GO111MODULE=auto

COPY . .

ARG SHORT_COMMIT_HASH

RUN go build -ldflags="-X github.com/image-server/image-server/core.BuildTimestamp=`date -u '+%Y-%m-%d_%I:%M:%S%p_%z'` -X github.com/image-server/image-server/core.GitHash=${SHORT_COMMIT_HASH}"

FROM alpine:3.15

RUN apk add --no-cache imagemagick
RUN apk add --no-cache ca-certificates

WORKDIR /opt/image-server

RUN mkdir -p public
RUN chmod 775 -R public

COPY start.sh .

RUN mkdir -p bin
COPY --from=0 /go/src/github.com/image-server/image-server/image-server bin/image-server
RUN chmod 775 -R bin/image-server

COPY sample_images/public/ /opt/image-server/sample
RUN chmod -R +r /opt/image-server/sample

EXPOSE 7000
EXPOSE 7002

CMD ["./start.sh"]
