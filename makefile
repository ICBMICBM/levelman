.PHONY: clean
clean:
	rm levelman
build:
	go build -ldflags="-X main.Commit=$(git rev-parse HEAD)"