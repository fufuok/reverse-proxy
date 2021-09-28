.PHONY: build clean install start help

PROJECT=rproxy
CI_BIN=bin/$(PROJECT)
OPT_DIR=/opt/xunyou/$(PROJECT)
OPT_BIN=$(OPT_DIR)/bin
OPT_LOG=$(OPT_DIR)/log

all: build

build:
	go build -v -o $(CI_BIN) cli/$(PROJECT)/main.go

clean:
	killall $(PROJECT) || true
	rm $(CI_BIN)

install:
	mkdir -p $(DESTDIR)/$(OPT_DIR)/bin
	mkdir -p $(DESTDIR)/$(OPT_DIR)/log
	install -m 755 -D $(CI_BIN) $(DESTDIR)/$(OPT_DIR)/bin/$(PROJECT)

start:
	$(OPT_DIR)/bin/$(PROJECT)

help:
	@echo "make: compile packages and dependencies"
	@echo "make clean: stop service and remove object files"
	@echo "make install: deploy"
	@echo "make start: start service"
