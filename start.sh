#!/usr/bin/env sh

LOCAL_BASE_PATH=${LOCAL_BASE_PATH:-public}
UPLOADER=${UPLOADER:-aws}
AWS_S3_BUCKET=${AWS_S3_BUCKET:-image-server}
AWS_REGION=${AWS_REGION:-us-east-1}
SERVER_LISTEN=${SERVER_LISTEN:-0.0.0.0}
REMOTE_BASE_URL=${REMOTE_BASE_URL:-https://s3-${AWS_REGION}.amazonaws.com/${AWS_S3_BUCKET}}

exec bin/image-server --local_base_path ${LOCAL_BASE_PATH} --uploader ${UPLOADER} --aws_bucket ${AWS_S3_BUCKET} --aws_region ${AWS_REGION} --listen ${SERVER_LISTEN} --remote_base_url ${REMOTE_BASE_URL} server
