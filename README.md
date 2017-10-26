# logspout-gcs

Logspout-GCS is a [logspout](https://github.com/gliderlabs/logspout) module that
streams docker application logs to Google Cloud Storage at some FLUSH_INTERVAL or MAX_SINK_SIZE_MB.

Heavily inspired by [logspout-s3](https://github.com/pressly/logspout-s3) by Pressly.

## Usage

```shell
docker run -d --name=logspout-gcs \
	-e 'BACKLOG=false' \
	-e 'GCS_KEY_FILE=YOUR_KEY_FILE' \
	-e 'FLUSH_INTERVAL=120' \
	-e 'MAX_SINK_SIZE_MB=16' \
	--volume=/var/run/docker.sock:/var/run/docker.sock \
	localyyz/logspout-gcs \
	gcs://localyyz?path=/logs
```

The first argument is parsed with the standard `url` library,
example output of above would be:

```go
    r := &Route{
        Address: 'localyyz,
        Adapter: 'gcs',
        Options: map{string}string{
            'path': '/logs',
        },
    }
```

## Image run options

The container app supports a few environment variables as options:

* `GCS_KEY_FILE` : your GCS service account JSON key file (optional: if you're using Google App Engine or Google Compute Engine you don't need this)
* `FLUSH_INTERVAL` : interval that collected logs are then uploaded to GCS, as seconds (default: 120)
* `MAX_SINK_SIZE_MB` : max buffer size for log collection before sending to GCS, as MB (default: 16)

## LICENSE

MIT
