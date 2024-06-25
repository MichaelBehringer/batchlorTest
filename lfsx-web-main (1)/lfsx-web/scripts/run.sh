#!/bin/sh

# Some color definitions
GREEN='\033[0;32m'
NC='\033[0m'

# Environment variables that the controller does need
export APP_JWT_PATH="./key.txt"
export LOGGER_PRINTLEVEL="TRACE"
export LOGGER_LEVEL="TRACE"
export APP_DEV_USE_DEVSERVER="true"
# Use loale VNC address instead of querieng the kubernetes API #
export APP_DEV_VNC_ADDRESS="localhost:5910"
export APP_DEV_GUACAMOL_ADDRESS="localhost:4822"
# ----- #
export APP_PRODUCTION="false"
export APP_LFS_SERVICE_ENDPOINT="https://webapi.hama.com/lfstest/"
export APP_LFS_SERVICE_ENDPOINT_JWT_NAME="cookie-javalfs"
export APP_ADDRESS="0.0.0.0:4020"
export KUBERNETES_NAMESPACE="lfsx-web-test"
export APP_LFS_IMAGE_NAME="containers.hama.de/registry-hama/lfsx-web-lfs:11.51.32-SNAPSHOT-snapshot-b4201a52a35184a567cd6aa108a3e792e18fca7c0"

export APP_LFS_PROC_PATH="/opt/lfsx/lfsx"
export APP_LFS_PROC_DATA="/opt/lfs-user"
export APP_LFS_CONFIG="/opt/lfs-user/config"

# Configuration options for the LFS go application
export APP_LFS_ADDRESS="0.0.0.0:4021"

# Determine project to use withing go workspace
path="./controller"
app="lfsx-web-controller"
if [ "$1" = "lfs" ]; then
    path="./lfs"
    app="lfs-web-lfs"
fi

nodemon --delay 1s -e go,html,yaml --ignore ""$path"/web/app/" --signal SIGTERM --quiet --exec \
'echo -e "\n'"$GREEN"'[Restarting]'"$NC"'" && go run -ldflags "-X main.version="$(cat VERSION)"" '"$path"'/cmd/'"$app" -- "$@" "|| exit 1"