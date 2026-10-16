#!/bin/sh
# Install commitheat from GitHub Releases.
#
#   curl -fsSL https://raw.githubusercontent.com/zori7/commitheat/main/install.sh | sh
#
# Environment variables:
#   COMMITHEAT_VERSION      release tag to install, e.g. v0.1.0 (default: latest)
#   COMMITHEAT_INSTALL_DIR  where to put the binary
#                           (default: /usr/local/bin if writable, else ~/.local/bin)
set -eu

repo="zori7/commitheat"
version=${COMMITHEAT_VERSION:-latest}

say() { printf 'commitheat: %s\n' "$*"; }
die() {
	printf 'commitheat: error: %s\n' "$*" >&2
	exit 1
}

detect_platform() {
	case $(uname -s) in
	Darwin) os=darwin ;;
	Linux) os=linux ;;
	*) die "unsupported OS: $(uname -s) (supported: macOS, Linux)" ;;
	esac
	case $(uname -m) in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*) die "unsupported CPU architecture: $(uname -m) (supported: x86_64, arm64)" ;;
	esac
}

download() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL -o "$2" "$1"
	elif command -v wget >/dev/null 2>&1; then
		wget -q -O "$2" "$1"
	else
		die "curl or wget is required"
	fi
}

sha256() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		die "sha256sum or shasum is required to verify the download"
	fi
}

choose_install_dir() {
	if [ -n "${COMMITHEAT_INSTALL_DIR:-}" ]; then
		install_dir=$COMMITHEAT_INSTALL_DIR
	elif [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
		install_dir=/usr/local/bin
	else
		install_dir="$HOME/.local/bin"
	fi
	mkdir -p "$install_dir" || die "cannot create $install_dir"
	[ -w "$install_dir" ] || die "$install_dir is not writable; set COMMITHEAT_INSTALL_DIR to another directory"
}

main() {
	detect_platform
	asset="commitheat_${os}_${arch}.tar.gz"

	# COMMITHEAT_BASE_URL is for testing against a local build of the release files.
	if [ -n "${COMMITHEAT_BASE_URL:-}" ]; then
		base=$COMMITHEAT_BASE_URL
	elif [ "$version" = latest ]; then
		base="https://github.com/$repo/releases/latest/download"
	else
		base="https://github.com/$repo/releases/download/$version"
	fi

	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT INT TERM

	say "downloading $asset ($version)"
	download "$base/$asset" "$tmp/$asset" || die "download failed: $base/$asset"
	download "$base/checksums.txt" "$tmp/checksums.txt" || die "download failed: $base/checksums.txt"

	expected=$(awk -v f="$asset" '$2 == f || $2 == "*" f { print $1 }' "$tmp/checksums.txt")
	[ -n "$expected" ] || die "no checksum for $asset in checksums.txt"
	actual=$(sha256 "$tmp/$asset")
	[ "$expected" = "$actual" ] || die "checksum mismatch for $asset (expected $expected, got $actual)"

	tar -xzf "$tmp/$asset" -C "$tmp" commitheat || die "could not extract $asset"

	choose_install_dir
	# Install via a temp name and rename, so a running copy is never half-written.
	cp "$tmp/commitheat" "$install_dir/.commitheat.tmp"
	chmod 755 "$install_dir/.commitheat.tmp"
	mv -f "$install_dir/.commitheat.tmp" "$install_dir/commitheat"

	say "installed $("$install_dir/commitheat" --version 2>/dev/null || echo commitheat) to $install_dir/commitheat"

	case ":$PATH:" in
	*":$install_dir:"*) ;;
	*)
		say "$install_dir is not on your PATH. Add it with:"
		printf '\n    export PATH="%s:$PATH"\n\n' "$install_dir"
		;;
	esac
}

main
