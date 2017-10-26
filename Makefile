build:
	@echo "hi"

docker-image:
	export VERSION=gcs && docker build -t localyyz/logspout-gcs .
