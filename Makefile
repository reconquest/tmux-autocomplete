RELEASE = alpha

VERSION = $(shell printf "%s.%s.%s" \
	$$(git rev-list --count HEAD) \
	$(RELEASE) \
	$$(git rev-parse --short HEAD) \
)

LICENSE_PUBLIC_KEY = $(shell echo 123)

NAME = $(notdir $(PWD))

DESCRIPTION = Autocompletion system for tmux multiplexer

FPM := --force \
	--maintainer "reconquest@gitlab" \
	--input-type dir \
	--name tmux-autocomplete \
	--version $(VERSION) \
	--description "$(DESCRIPTION)" \
	--log error \
	usr/

build:
	@echo '> Building version $(VERSION)'
	@go build \
		-ldflags="-X=main.version=$(VERSION) \
			-X=main.licensePublicKey=$(LICENSE_PUBLIC_KEY)" \
		$(GCFLAGS)

pkg/tree: build
	@rm -rf pkg/tree
	@mkdir -p pkg/tree/usr/bin/ pkg/tree/usr/share/tmux-autocomplete/themes/
	@cp -r share/themes pkg/tree/usr/share/tmux-autocomplete/
	@cp tmux-autocomplete pkg/tree/usr/bin/
	@cp tmux-autocomplete pkg/tree/usr/bin/
	@cp share/tmux-autocomplete-url pkg/tree/usr/bin/

pkg/tree_osx: build
	@rm -rf pkg/tree_osx
	@mkdir -p pkg/tree_osx/usr/local/bin/ pkg/tree_osx/usr/local/share/tmux-autocomplete/themes/
	@cp -r share/themes pkg/tree_osx/usr/local/share/tmux-autocomplete/
	@cp tmux-autocomplete pkg/tree_osx/usr/local/bin/
	@cp share/tmux-autocomplete-url pkg/tree_osx/usr/local/bin/

pkg_arch: pkg/tree
	@echo '> Building Arch Linux package'
	@fpm -t pacman -p pkg/tmux-autocomplete_VERSION_ARCH.pkg.tar.xz -C pkg/tree $(FPM)

pkg_deb: pkg/tree
	@echo '> Building Debian package'
	@fpm -t deb -p pkg/tmux-autocomplete_VERSION_ARCH.deb -C pkg/tree $(FPM)

pkg_rpm: pkg/tree
	@echo '> Building RPM package'
	@fpm -t rpm -p pkg/tmux-autocomplete_VERSION_ARCH.rpm -C pkg/tree $(FPM)

pkg_tar: pkg/tree
	@echo '> Building TAR package'
	@fpm -t tar -p pkg/tmux-autocomplete_VERSION_ARCH.tar -C pkg/tree $(FPM)

pkg_osx: pkg/tree_osx
	@echo '> Building OSX package'
	@fpm -t osxpkg -p pkg/tmux-autocomplete_VERSION_ARCH.pkg \
		--osxpkg-identifier-prefix com.gitlab.reconquest \
		-C pkg/tree_osx \
		$(FPM)

.PHONY: pkg
pkg: pkg_arch pkg_deb pkg_rpm pkg_tar

license/$(RELEASE).private:
	lkgen gen -o license/$(RELEASE).private

lkpub: license/$(RELEASE).private
	lkgen pub -o license/$(RELEASE).public license/$(RELEASE).private
