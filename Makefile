.PHONY: all docker
all: docker

../go-base/Dockerfile:
	(cd ..; git clone https://github.com/dgl/go-base)

docker: Dockerfile
	DOCKER_BUILDKIT=1 docker build .

docker-debug: Dockerfile
	DOCKER_BUILDKIT=1 docker build --build-arg BUILD_DEBUG=1 --progress plain .

Dockerfile: ../go-base/Dockerfile Dockerfile.tail
	cat $^ > $@

redisircd: */*.go go.mod go.sum
	go build ./cmd/redisircd

run: redisircd
	./$^
