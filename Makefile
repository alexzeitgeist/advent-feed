BINARY = galaxus-advent-rss
INSTALL_PATH = /usr/local/bin

.PHONY: build clean install uninstall

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) .

clean:
	rm -f $(BINARY)

install: build
	install -m 755 $(BINARY) $(INSTALL_PATH)/$(BINARY)

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY)
