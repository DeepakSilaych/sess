.PHONY: build agents test integration install uninstall clean
PREFIX ?= $(HOME)/.local

agents:
	./scripts/build-agents.sh

build: agents
	go build -trimpath -ldflags='-s -w' -o dist/sess ./cmd/sess

test:
	go test -race ./...
	go vet ./...

integration: build
	python3 test/integration/run.py

install: build
	install -d "$(DESTDIR)$(PREFIX)/bin"
	install -m 755 dist/sess "$(DESTDIR)$(PREFIX)/bin/sess"
	install -d "$(DESTDIR)$(PREFIX)/share/bash-completion/completions" "$(DESTDIR)$(PREFIX)/share/zsh/site-functions"
	dist/sess completion bash > "$(DESTDIR)$(PREFIX)/share/bash-completion/completions/sess"
	dist/sess completion zsh > "$(DESTDIR)$(PREFIX)/share/zsh/site-functions/_sess"

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)/bin/sess" "$(DESTDIR)$(PREFIX)/share/bash-completion/completions/sess" "$(DESTDIR)$(PREFIX)/share/zsh/site-functions/_sess"

clean:
	rm -rf dist
	rm -f internal/provision/assets/*.gz
