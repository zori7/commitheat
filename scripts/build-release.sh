#!/bin/sh
# Build release archives for every supported platform into dist/.
#
#   scripts/build-release.sh v0.1.0
#
# Produces dist/commitheat_<os>_<arch>.tar.gz and dist/checksums.txt,
# the layout install.sh expects. Upload them with:
#
#   gh release create v0.1.0 dist/* --generate-notes
set -eu

version=${1:?usage: scripts/build-release.sh <version>}
targets="darwin/amd64 darwin/arm64 linux/amd64 linux/arm64"

cd "$(dirname "$0")/.."
rm -rf dist
mkdir -p dist

for target in $targets; do
	os=${target%/*}
	arch=${target#*/}
	name="commitheat_${os}_${arch}"
	echo "building $name"
	mkdir -p "dist/$name"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
		-ldflags "-s -w -X commitheat/cmd.Version=$version" \
		-o "dist/$name/commitheat" .
	cp LICENSE README.md "dist/$name/"
	tar -czf "dist/$name.tar.gz" -C "dist/$name" commitheat LICENSE README.md
	rm -rf "dist/$name"
done

cd dist
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum -- *.tar.gz >checksums.txt
else
	shasum -a 256 -- *.tar.gz >checksums.txt
fi
cat checksums.txt
