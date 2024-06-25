#!/bin/bash

printError() {
    echo "Failed to build app. See previous output for more informations" 1>&2
}

set -e
trap printError EXIT

# Get Version of the program
version="$(cat VERSION)"

# Build controller
cd controller
if [ "$1" != "--noWeb" ]; then
    # Install dependencies (--prefix does not work under windows -> cd)
    cd ./web/app
    npm install --save-dev

    # Build frontend
    npm run build

    cd ../../
fi

# Build go binary with all assets
flags="-X main.version="$version""
GOOS=linux GOARCH=amd64 go build -o "../lfsx-web-controller-"$version"-amd64" -ldflags "$flags" ./cmd/lfsx-web-controller
#CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o "replica-"$version"-amd64.exe" -ldflags "$flags" ./cmd/replicad
cd ..

echo "Build finished"
trap - EXIT
exit 0