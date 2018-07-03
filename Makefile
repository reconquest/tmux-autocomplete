VERSION = $(shell printf "%s.%s" \
	$$(git rev-list --count HEAD) \
	$$(git rev-parse --short HEAD) \
)

NAME = $(notdir $(PWD))

DESCRIPTION = Autocompletion system for tmux multiplexer

FPM := --force \
	--maintainer "reconquest@github" \
	--input-type dir \
	--name $(NAME) \
	--version $(VERSION) \
	--description "$(DESCRIPTION)" \
	--chdir pkg/ \
	--log error \
	usr/

build:
	@echo '> Building version $(VERSION)'
	@go build -ldflags="-X=main.version=$(VERSION)" $(GCFLAGS)

pkg/tree: build
	@rm -rf pkg/tree
	@mkdir -p pkg/tree/usr/bin/ pkg/tree/usr/share/tmux-autocomplete/themes/
	@cp -r themes pkg/tree/usr/share/tmux-autocomplete/
	@cp $(NAME) pkg/tree/usr/bin/

pkg_arch: pkg/tree
	@echo '> Building Arch Linux package'
	@fpm -t pacman -p pkg/tmux-autocomplete_VERSION_ARCH.pkg.tar.xz $(FPM)

pkg_deb: pkg/tree
	@echo '> Building Debian package'
	@fpm -t deb -p pkg/tmux-autocomplete_VERSION_ARCH.deb $(FPM)

pkg_rpm: pkg/tree
	@echo '> Building RPM package'
	@fpm -t rpm -p pkg/tmux-autocomplete_VERSION_ARCH.rpm $(FPM)

pkg_tar: pkg/tree
	@echo '> Building TAR package'
	@fpm -t tar -p pkg/tmux-autocomplete_VERSION_ARCH.tar $(FPM)

pkg_osx:
	@echo '> Building OSX package'
	@fpm -t osxpkg -p pkg/tmux-autocomplete_VERSION_ARCH.dmg $(FPM)

.PHONY: pkg
pkg: pkg/tree pkg_arch pkg_deb pkg_rpm pkg_tar