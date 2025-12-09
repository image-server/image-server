FROM golang:1.25-alpine

# Install build dependencies for govips (CGO)
RUN apk add --no-cache build-base vips-dev pkgconfig

WORKDIR ${GOPATH}/src/github.com/image-server/image-server

ENV GO111MODULE=auto
ENV CGO_ENABLED=1

COPY . .

ARG SHORT_COMMIT_HASH

RUN go build -ldflags="-X github.com/image-server/image-server/core.BuildTimestamp=`date -u '+%Y-%m-%d_%I:%M:%S%p_%z'` -X github.com/image-server/image-server/core.GitHash=${SHORT_COMMIT_HASH}"

FROM alpine:3.21

# Install vips for image processing (poppler for PDF, heif for iPhone images)
RUN apk add --no-cache vips vips-poppler vips-heif
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
