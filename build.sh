#!/usr/bin/env bash

# call with VERSION variable set (ie, VERSION=0.5.0 ./build.sh)

set -e
set -u

platforms=(
  "linux/amd64"
  "linux/arm"
  "linux/386"
  "freebsd/amd64"
  "freebsd/arm"
  "freebsd/386"
  "windows/amd64"
  "windows/386"
)

for platform in "${platforms[@]}" ; do
  platform_split=(${platform//\// })
  GOOS=${platform_split[0]}
  GOARCH=${platform_split[1]}

  output_name="fscanary_${VERSION}_${GOOS}_${GOARCH}"
  [ $GOOS = "windows" ] && output_name+='.exe'

  echo "Building $output_name"
  env GOOS=$GOOS GOARCH=$GOARCH \
    go build -ldflags "-X main.version=$VERSION" -o "$output_name"

  tarball_name="fscanary_${VERSION}_${GOOS}_${GOARCH}.tar.bz2"
  echo "Creating tarball $tarball_name"
  tar cjf release/"$tarball_name" \
    --owner=root --group=root \
    "$output_name" \
    fscanary.conf.sample \
    init/
done

cd release
sha256sum fscanary_${VERSION}_*_*.tar.bz2 > SHA256SUMS
cat SHA256SUMS
