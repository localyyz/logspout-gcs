build:
	@mkdir -p ./bin
	GOGC=off go build -i -o ./bin/ex-server .

docker-image:
	docker build -t logspout-gcs-ex-server .

docker-run:
	docker run --rm -p 3333:3333 logspout-gcs-ex-server

docker-run-logspout:
	docker run --rm --name=logspout-gcs \
		-v '${KEY_FILE}':'/data/key.json' \
		-e 'BACKLOG=false' \
		-e 'GCS_KEY_FILE=/data/key.json' \
		-e 'FLUSH_INTERVAL=2' \
		-e 'MAX_SINK_SIZE_MB=1' \
		--volume=/var/run/docker.sock:/var/run/docker.sock \
		--publish=127.0.0.1:8000:80 \
		localyyz/logspout-gcs \
		gcs://localyyz?path=logs
