
DOCKERREPO       := endianogino/netperf:1.1

all: nptests container push

nptests:
	docker run --rm -v $(shell pwd):/go/src/github.com/mrahbar/k8s-nptest -w /go/src/github.com/mrahbar/k8s-nptest -e CGO_ENABLED=0 golang:1.8.3-stretch go build -a -installsuffix cgo -o nptests main.go

container: nptests
	mkdir -p Dockerbuild && \
	cp -f nptests Dockerbuild/ &&\
	docker build -t $(DOCKERREPO) Dockerbuild/ 

push: container
	docker push $(DOCKERREPO)