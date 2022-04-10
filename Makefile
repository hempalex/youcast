BINARY=youcast
SRC=youcast.go

PLATFORMS=darwin freebsd
ARCHITECTURES=amd64 arm64

all: build

build:
	go build -o bin/$(BINARY) $(SRC)

run:
	go run $(SRC)

release:
	$(foreach GOOS, $(PLATFORMS), \
		$(foreach GOARCH, $(ARCHITECTURES), \
			$(shell \
				export GOOS=$(GOOS); \
				export GOARCH=$(GOARCH); \
				go build -o bin/$(BINARY)-$(GOOS)-$(GOARCH) $(SRC) \
			) \
		) \
	)

clean:
	rm -f bin/$(BINARY)-*

