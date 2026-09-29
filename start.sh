#!/usr/bin/env sh

LOCAL_BASE_PATH=${LOCAL_BASE_PATH:-public}
UPLOADER=${UPLOADER:-aws}
AWS_S3_BUCKET=${AWS_S3_BUCKET:-image-server}
AWS_REGION=${AWS_REGION:-us-east-1}
SERVER_LISTEN=${SERVER_LISTEN:-0.0.0.0}
# Remote store default only for S3. Other uploaders (noop) have no remote copy:
# leaving it empty makes the server skip remote lookups for missing files.
if [ "$UPLOADER" = "aws" ]; then
  REMOTE_BASE_URL=${REMOTE_BASE_URL:-https://s3-${AWS_REGION}.amazonaws.com/${AWS_S3_BUCKET}}
fi

mkdir -p /opt/image-server/public
# Values quoted so an empty one cannot swallow the next flag; CUSTOM_FLAGS is word-split on purpose.
exec bin/image-server server --local_base_path "${LOCAL_BASE_PATH}" --uploader "${UPLOADER}" --aws_bucket "${AWS_S3_BUCKET}" --aws_region "${AWS_REGION}" --listen "${SERVER_LISTEN}" --remote_base_url "${REMOTE_BASE_URL:-}" --extensions "${ALLOWED_EXTENSIONS}" ${CUSTOM_FLAGS}
