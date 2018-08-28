NAME = tmux-autocomplete
DESCRIPTION = Autocompletion system for tmux multiplexer

RELEASE = $(shell git describe --tags --abbrev=0)
RELEASE = alpha
VERSION = $(shell git describe --tags | sed 's/\-/./g')

define LICENSE_PUBLIC_KEY
$(shell cat license/$(RELEASE).public)
endef

FPM := --force \
	--url "https://tmux.reconquest.io/" \
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
	@echo ':: Building version $(VERSION)'
	@go build \
		-ldflags="-X=main.version=$(VERSION) \
			-X main.release=$(RELEASE) \
			-X=main.licensePublicKey=$(call LICENSE_PUBLIC_KEY)" \
		$(GCFLAGS)

pkg_tree/linux: build
	@rm -rf pkg_tree/linux
	@mkdir -p pkg_tree/linux/usr/bin/ pkg_tree/linux/usr/share/tmux-autocomplete/themes/
	@cp -r share/themes pkg_tree/linux/usr/share/tmux-autocomplete/
	@cp tmux-autocomplete pkg_tree/linux/usr/bin/
	@cp tmux-autocomplete pkg_tree/linux/usr/bin/
	@cp share/tmux-autocomplete-url pkg_tree/linux/usr/bin/

pkg_tree/osx: build
	@rm -rf pkg_tree/osx
	@mkdir -p pkg_tree/osx/usr/local/bin/ pkg_tree/osx/usr/local/share/tmux-autocomplete/themes/
	@cp -r share/themes pkg_tree/osx/usr/local/share/tmux-autocomplete/
	@cp tmux-autocomplete pkg_tree/osx/usr/local/bin/
	@cp share/tmux-autocomplete-url pkg_tree/osx/usr/local/bin/

pkg_arch: pkg_tree/linux
	@echo ':: Building Arch Linux package'
	@mkdir -p pkg/arch/
	@fpm -t pacman -p pkg/arch/tmux-autocomplete_$(VERSION).pkg.tar.xz -C pkg_tree/linux $(FPM)

pkg_deb: pkg_tree/linux
	@echo ':: Building Debian package'
	@mkdir -p pkg/deb/
	@fpm -t deb -p pkg/deb/tmux-autocomplete_$(VERSION).deb -C pkg_tree/linux $(FPM)

pkg_rpm: pkg_tree/linux
	@echo ':: Building RPM package'
	@mkdir -p pkg/rpm/
	@fpm -t rpm -p pkg/rpm/tmux-autocomplete_$(VERSION).rpm -C pkg_tree/linux $(FPM)

pkg_tar: pkg_tree/linux
	@echo ':: Building TAR package'
	@mkdir -p pkg/tar/
	@fpm -t tar -p pkg/tar/tmux-autocomplete_$(VERSION).tar -C pkg_tree/linux $(FPM)

pkg_osx: pkg_tree/osx
	@echo ':: Building OSX package'
	@mkdir -p pkg/osx/
	@fpm -t osxpkg -p pkg/osx/tmux-autocomplete_$(VERSION).pkg \
		--osxpkg-identifier-prefix com.gitlab.reconquest \
		-C pkg_tree/osx \
		$(FPM)

.PHONY: pkg
pkg: pkg_arch pkg_deb pkg_rpm pkg_tar

release: pkg
	@echo ":: Building && installing package on OSX"
	./osx-install
	@echo ":: Downloading OSX package to local directory"
	./osx-package
	@echo ":: Uploading new archives to remote host"
	@./upload

license/$(RELEASE).private:
	lkgen gen -o license/$(RELEASE).private

license/$(RELEASE).public: license/$(RELEASE).private
	lkgen pub -o license/$(RELEASE).public license/$(RELEASE).private
