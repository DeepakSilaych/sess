.PHONY: install uninstall clean test

PREFIX ?= /usr/local
SESS_DIR ?= $(HOME)/.sess

BIN_DIR = $(DESTDIR)$(PREFIX)/bin
COMPLETION_DIR_BASH = $(DESTDIR)$(PREFIX)/share/bash-completion/completions
COMPLETION_DIR_ZSH = $(DESTDIR)$(PREFIX)/share/zsh/site-functions

test:
	@echo "Running sess tests..."
	@bash -n bin/sess && echo "✓ sess: syntax OK" || exit 1
	@bash -n etc/bash-completion/sess && echo "✓ bash completion: OK" || exit 1
	@bash -c 'source etc/bash-completion/sess' && echo "✓ bash completion loads OK" || exit 1
	@echo "Done."

install: bin/sess
	@mkdir -p $(BIN_DIR)
	@mkdir -p $(COMPLETION_DIR_BASH)
	@mkdir -p $(COMPLETION_DIR_ZSH)
	cp bin/sess $(BIN_DIR)/sess
	chmod +x $(BIN_DIR)/sess
	cp etc/bash-completion/sess $(COMPLETION_DIR_BASH)/sess
	cp etc/zsh-completion/_sess $(COMPLETION_DIR_ZSH)/_sess
	@echo "Installed to $(PREFIX)"
	@echo "  sess         → $(BIN_DIR)/sess"
	@echo "  bash comp    → $(COMPLETION_DIR_BASH)/sess"
	@echo "  zsh comp     → $(COMPLETION_DIR_ZSH)/_sess"

uninstall:
	rm -f $(BIN_DIR)/sess
	rm -f $(COMPLETION_DIR_BASH)/sess
	rm -f $(COMPLETION_DIR_ZSH)/_sess
	@echo "Uninstalled from $(PREFIX)"

clean:
	rm -rf node_modules