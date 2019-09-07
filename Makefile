VERSION := $(shell git describe --tags --dirty --always)
LDFLAGS := '-s -w -extldflags "-static" -X main.Version=${VERSION}'

static:
	CGO_ENABLED=0 go build -trimpath -ldflags=$(LDFLAGS) -a -o luxtronik2-exporter .

container:
	docker build -t shorez/luxtronik2-exporter .
