#!/usr/bin/env sh

BASE_PATH=${BASE_PATH:-image-server/workspace}
UPLOADER=${UPLOADER:-aws}
AWS_S3_BUCKET=${AWS_S3_BUCKET:-image-server}
AWS_REGION=${AWS_REGION:-us-east-1}
SERVER_LISTEN=${SERVER_LISTEN:-0.0.0.0}
IMAGE_SERVER_REMOTE_BASE_URL=${IMAGE_SERVER_REMOTE_BASE_URL:-https://s3-${AWS_REGION}.amazonaws.com/${AWS_S3_BUCKET}}

bin/image-server --local_base_path ${BASE_PATH} --uploader ${UPLOADER} --aws_bucket ${AWS_S3_BUCKET} --aws_region ${AWS_REGION} --listen ${SERVER_LISTEN} --remote_base_url ${IMAGE_SERVER_REMOTE_BASE_URL} server