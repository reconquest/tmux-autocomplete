NAME = tmux-autocomplete
DESCRIPTION = Autocompletion system for tmux multiplexer

RELEASE = $(shell git describe --tags --abbrev=0)
VERSION = $(shell git describe --tags)

define LICENSE_PUBLIC_KEY
$(shell cat license/$(RELEASE).public)
endef

FPM := --force \
	--maintainer "reconquest@gitlab" \
	--input-type dir \
	--name tmux-autocomplete \
	--version "$(VERSION)" \
	--description "$(DESCRIPTION)" \
	--log error \
	usr/

version:
	@echo $(VERSION)

build: license/$(RELEASE).private
	@echo '> Building version $(VERSION)'
	@go build \
		-ldflags="-X=main.version=$(VERSION) \
			-X main.release=$(RELEASE) \
			-X=main.licensePublicKey=$(call LICENSE_PUBLIC_KEY)" \
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
	@mkdir pkg/arch/
	@fpm -t pacman -p pkg/arch/tmux-autocomplete_VERSION_ARCH.pkg.tar.xz -C pkg/tree $(FPM)

pkg_deb: pkg/tree
	@echo '> Building Debian package'
	@mkdir pkg/deb/
	@fpm -t deb -p pkg/deb/tmux-autocomplete_VERSION_ARCH.deb -C pkg/tree $(FPM)

pkg_rpm: pkg/tree
	@echo '> Building RPM package'
	@mkdir pkg/rpm/
	@fpm -t rpm -p pkg/rpm/tmux-autocomplete_VERSION_ARCH.rpm -C pkg/tree $(FPM)

pkg_tar: pkg/tree
	@echo '> Building TAR package'
	@mkdir pkg/rpm/
	@fpm -t tar -p pkg/tar/tmux-autocomplete_VERSION_ARCH.tar -C pkg/tree $(FPM)

pkg_osx: pkg/tree_osx
	@echo '> Building OSX package'
	@fpm -t osxpkg -p pkg/osx/tmux-autocomplete_VERSION_ARCH.pkg \
		--osxpkg-identifier-prefix com.gitlab.reconquest \
		-C pkg/tree_osx \
		$(FPM)

.PHONY: pkg
pkg: pkg_arch pkg_deb pkg_rpm pkg_tar

license/$(RELEASE).private:
	lkgen gen -o license/$(RELEASE).private

license/$(RELEASE).public: license/$(RELEASE).private
	lkgen pub -o license/$(RELEASE).public license/$(RELEASE).private
