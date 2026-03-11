APP_NAME = wallgtk
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin

.PHONY: build install uninstall clean run

build:
	go build -o $(APP_NAME)

run:
	go run .

install: build
	./install.sh

uninstall:
	./uninstall.sh

clean:
	rm -f $(APP_NAME)
