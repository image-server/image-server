# Image Server

A high-performance image processing server written in Go. Supports on-demand image resizing, format conversion, and cloud storage integration.

## Features

- **On-demand image processing** - Resize, crop, and convert images via URL parameters
- **Multiple output formats** - JPEG, WebP, GIF, PNG, HEIC/HEIF (iPhone images)
- **Cloud storage** - Upload processed images to Amazon S3
- **Signed URLs** - Secure uploads with HMAC-SHA256 signed URLs (similar to AWS S3 pre-signed URLs)
- **Batch processing** - Process multiple image sizes in a single request
- **Prometheus metrics** - Built-in metrics endpoint for monitoring
- **Webhooks** - Notify external systems when images are uploaded or processed
- **Docker support** - Ready-to-use Docker image

## Quick Start

### Using Docker

```bash
docker build -t image-server .
docker run -p 7000:7000 -p 7002:7002 image-server
```

### Building from Source

Requires Go 1.21+ and libvips.

```bash
# macOS
brew install vips

# Ubuntu/Debian
apt-get install libvips-dev

# Build
go build -o image-server .

# Run
./image-server server --port 7000
```

## Server

### Uploading Images

Images are uploaded to a namespace. Namespaces group image types (e.g., avatars vs product images may need different sizes).

**Upload from URL:**
```bash
curl -X POST "http://localhost:7000/products?source=https://example.com/image.jpg"
```

**Upload binary data:**
```bash
curl --data-binary "@./image.jpg" -X POST http://localhost:7000/products
```

**Response:**
```json
{
  "hash": "6e0072682e66287b662827da75b244a3",
  "height": 600,
  "width": 800,
  "content_type": "image/jpeg"
}
```

**Upload and process immediately:**
```bash
curl --data-binary "@./image.jpg" -X POST "http://localhost:7000/products?outputs=x300.jpg,x300.webp"
```

### Retrieving Images

Images are accessed via their hash, partitioned into path segments:

```
GET http://localhost:7000/{namespace}/{hash[0:3]}/{hash[3:6]}/{hash[6:9]}/{hash[9:]}/{dimensions}.{format}
```

**Examples:**

```bash
# By width (maintains aspect ratio)
GET http://localhost:7000/products/6e0/072/682/e66287b662827da75b244a3/w200.jpg

# Square crop
GET http://localhost:7000/products/6e0/072/682/e66287b662827da75b244a3/x200.jpg

# Specific dimensions (width x height)
GET http://localhost:7000/products/6e0/072/682/e66287b662827da75b244a3/300x200.jpg

# With quality adjustment (1-100)
GET http://localhost:7000/products/6e0/072/682/e66287b662827da75b244a3/x200-q50.jpg

# WebP format
GET http://localhost:7000/products/6e0/072/682/e66287b662827da75b244a3/x200.webp
```

### Image Information

```bash
curl http://localhost:7000/products/6e0/072/682/e66287b662827da75b244a3/info.json
```

### Batch Processing

Process multiple sizes for an existing image:

```bash
curl -X POST "http://localhost:7000/products/6e0/072/682/e66287b662827da75b244a3/process?outputs=x100.jpg,x200.jpg,x300.webp"
```

## Signed URLs (Authentication)

Secure your image server by requiring signed URLs for uploads (and optionally reads). This works similarly to AWS S3 pre-signed URLs.

### Setup

1. **Generate a signing secret:**
   ```bash
   ./image-server generate-secret > /etc/image-server/secrets.txt
   ```

2. **Start the server with signature validation:**
   ```bash
   ./image-server server \
     --require-signature \
     --signing-secrets-file /etc/image-server/secrets.txt \
     --signature-max-ttl 60
   ```

3. **Generate signed URLs in your backend application:**

   The signature algorithm:
   ```
   StringToSign = METHOD + "\n" + PATH + "\n" + EXPIRES_UNIX_TIMESTAMP
   Signature = Base64RawURL(HMAC-SHA256(secret, StringToSign))
   ```

   URL format:
   ```
   POST /namespace?X-Expires=1702156800&X-Path=/namespace&X-Signature=...
   ```

   Go example:
   ```go
   import "github.com/image-server/image-server/core/signature"

   signer := signature.NewSigner("your-secret", "https://images.example.com")
   url := signer.SignURL("POST", "/products", 15*time.Minute)
   ```

4. **Test with the CLI:**
   ```bash
   ./image-server sign-url \
     --secret "your-secret" \
     --base-url "https://images.example.com" \
     --method POST \
     --path /products \
     --ttl 15m
   ```

### Configuration Options

| Flag | Description | Default |
|------|-------------|---------|
| `--require-signature` | Enable signature validation for uploads | `false` |
| `--require-signature-for-reads` | Also require signatures for GET requests | `false` |
| `--signing-secrets-file` | Path to file with secrets (one per line) | - |
| `--signature-max-ttl` | Maximum allowed TTL in minutes | `60` |

### Secret Rotation

The secrets file supports multiple secrets for rotation. Add new secrets to the top of the file:

```
new-secret-abc123
old-secret-xyz789
```

The server validates against all secrets, so you can:
1. Add the new secret
2. Update your backend apps to use it
3. Remove the old secret after existing URLs expire

### Path-Based Signing

Sign a path prefix to allow uploads to any path under it:

```bash
# Sign for entire namespace
./image-server sign-url --secret "..." --path /products --ttl 15m
# Allows: POST /products, POST /products/abc/def/...

# Sign for specific path only
./image-server sign-url --secret "..." --path /products/abc/def/ghi/jkl --ttl 15m
# Allows only that exact path
```

## Webhooks

Send HTTP notifications to external systems when images are uploaded or processed. Useful for triggering downstream workflows like OCR, ML pipelines, or cache invalidation.

### Setup

```bash
./image-server server \
  --webhook-url "https://api.example.com/webhooks/images" \
  --webhook-secret "$(./image-server generate-secret)" \
  --webhook-timeout 10 \
  --webhook-events "uploaded,batch_complete"
```

### Events

| Event | Trigger | Use Case |
|-------|---------|----------|
| `uploaded` | Original image uploaded to storage | Start processing pipeline |
| `processed` | Each image variant processed | Cache warming |
| `failed` | Processing error | Alerting |
| `batch_complete` | All variants done (or already existed) | Trigger downstream workflow |

By default, all events are sent. Use `--webhook-events` to filter.

### Payload Format

```json
{
  "event": "image.uploaded",
  "timestamp": "2025-12-09T15:30:00Z",
  "data": {
    "namespace": "products",
    "hash": "6e0072682e66287b662827da75b244a3",
    "width": 1920,
    "height": 1080,
    "content_type": "image/jpeg",
    "remote_url": "https://cdn.example.com/products/6e0/072/.../original"
  }
}
```

For `image.processed`:
```json
{
  "event": "image.processed",
  "timestamp": "2025-12-09T15:30:01Z",
  "data": {
    "namespace": "products",
    "hash": "6e0072682e66287b662827da75b244a3",
    "filename": "x300.webp",
    "format": "webp",
    "width": 300,
    "height": 169,
    "quality": 75,
    "remote_url": "https://cdn.example.com/products/6e0/072/.../x300.webp"
  }
}
```

### Security

Webhooks are signed with HMAC-SHA256. Verify the signature in your handler:

**Headers:**
```
X-Webhook-Signature: sha256=<hex-encoded-hmac>
X-Webhook-Timestamp: 1702135800
```

**Verification (Python):**
```python
import hmac
import hashlib

def verify_webhook(secret: str, timestamp: str, body: bytes, signature: str) -> bool:
    expected = "sha256=" + hmac.new(
        secret.encode(),
        f"{timestamp}.{body.decode()}".encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)

# In your handler:
if not verify_webhook(SECRET, request.headers["X-Webhook-Timestamp"],
                      request.body, request.headers["X-Webhook-Signature"]):
    return 401
```

**Verification (Go):**
```go
func verifyWebhook(secret, timestamp string, body []byte, signature string) bool {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write([]byte(fmt.Sprintf("%s.%s", timestamp, string(body))))
    expected := "sha256=" + hex.EncodeToString(h.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

### Configuration

| Flag | Description | Default |
|------|-------------|---------|
| `--webhook-url` | Endpoint URL (enables webhooks) | - |
| `--webhook-secret` | HMAC signing secret | - |
| `--webhook-timeout` | HTTP timeout in seconds | `10` |
| `--webhook-events` | Events to send (comma-separated) | all |

### Reliability

- Webhooks are sent asynchronously (non-blocking)
- Failed deliveries retry up to 3 times with exponential backoff (1s, 4s)
- Webhook failures don't affect image processing

## Cloud Storage (S3)

```bash
./image-server server \
  --uploader s3 \
  --aws_access_key_id $AWS_ACCESS_KEY_ID \
  --aws_secret_key $AWS_SECRET_KEY \
  --aws_bucket $AWS_BUCKET \
  --aws_region us-west-1 \
  --remote_base_path "images/" \
  --remote_base_url "https://cdn.example.com"
```

## Server Configuration

| Flag | Description | Default |
|------|-------------|---------|
| `--port` | Server port | `7000` |
| `--listen` | Listen address | `127.0.0.1` |
| `--local_base_path` | Local image storage directory | `public` |
| `--extensions` | Allowed file extensions | `jpg,gif,webp` |
| `--maximum_width` | Maximum output width | `1000` |
| `--default_quality` | Default JPEG/WebP quality | `75` |
| `--outputs` | Default output formats | - |
| `--uploader` | Storage backend (`s3` or `noop`) | auto |
| `--uploader_concurrency` | Parallel upload workers | `10` |
| `--processor_concurrency` | Parallel processing workers | `4` |
| `--http_timeout` | HTTP request timeout (seconds) | `5` |
| `--max_file_age` | Local file cleanup age (minutes) | `30` |

## Admin Server

A separate admin server runs on port 7002 with health and metrics endpoints:

| Endpoint | Description |
|----------|-------------|
| `/probe/ready` | Readiness check |
| `/probe/live` | Liveness check |
| `/metrics` | Prometheus metrics |

## CLI Commands

### Process images locally

```bash
./image-server cli /path/to/images --outputs "x300.jpg,x300.webp"
```

### Generate signing secret

```bash
./image-server generate-secret
./image-server generate-secret --length 64 --count 3
```

### Generate signed URL

```bash
./image-server sign-url --secret "..." --path /namespace --ttl 15m
```

### Version

```bash
./image-server version
```

## Monitoring

### Prometheus Metrics

Available at `http://localhost:7002/metrics`

### Statsd

```bash
./image-server server --enable_statsd --statsd_host 127.0.0.1 --statsd_port 8125
```

Events:
- `image_server.image_request` - Image processed and uploaded
- `image_server.image_request.{format}` - By format (jpg, webp, etc.)
- `image_server.image_request_fail` - Processing failed
- `image_server.original_downloaded` - Original fetched from source
- `image_server.original_unavailable` - Original not found (404)

### Profiling

```bash
./image-server server --profile
# pprof available at http://localhost:6060
```

## Development

### Running locally

```bash
# Without S3
make dev-server

# With S3
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_KEY=...
export AWS_BUCKET=...
export AWS_REGION=...
export IMG_REMOTE_BASE_PATH=...
export IMG_REMOTE_BASE_URL=...
make dev-server-s3
```

### Tests

```bash
make test
# or
go test ./...
```

### Building

```bash
make build
# Creates binaries in bin/ for multiple platforms
```

## Error Handling

| Status | Description |
|--------|-------------|
| 401 | Invalid or missing signature (when signatures required) |
| 404 | Image not found |
| 400 | Invalid request parameters |

## License

MIT
