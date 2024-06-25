# Build frontend
FROM node:18-alpine AS frontendBuilder

WORKDIR /build

# Copy repository files
COPY ./controller/web/app .

# Build
RUN npm install --save-dev \
    && npm run build


# Build app
FROM docker.io/golang:1.20-alpine3.17 AS builder
ARG VERSION="0.0.0"

WORKDIR /build

# Add certificates
COPY ./controller/docker/certs/* /etc/ssl/certs/ 
# First we need to be able to add certificates
RUN cat /etc/ssl/certs/hama-root.pem >> /etc/ssl/certs/ca-certificates.crt \
    && apk --no-cache add ca-certificates \
    && update-ca-certificates

# Add git with credentials to support private repositories in "go get"
# buildah bud --layers --network host --secret id=giteaApiKey,src=/home/ubuntugui/.secrets/gitea_api-key --tag=hama.de/lfsx-web-controller:0.0.0 --build-arg VERSION="0.0.0"  -f controller/docker/Dockerfile
ENV GIT_SSH_COMMAND="ssh -i /run/secrets/giteaSshKey -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"
RUN apk add --no-cache git openssh \
    && git config --global url."ssh://git@gitea.hama.de:".insteadOf "https://gitea.hama.de" \
    && go env -w GOPRIVATE="gitea.hama.de"

# Copy files
COPY ./ .

# Copy frontend files
COPY --from=frontendBuilder /build/dist/ ./controller/web/app/dist/

# Build
ENV GOINSECURE=proxy.golang.org
RUN --mount=type=secret,id=giteaSshKey  cd ./controller \
    && GOOS=linux GOARCH=amd64 go build -o "lfsx-web-controller-amd64" -ldflags "-X main.version=${VERSION}" ./cmd/lfsx-web-controller


FROM alpine:3.17
ARG ALPINE_VERSION=3.17

# Copy binary
RUN mkdir /app
COPY --from=builder /build/controller/lfsx-web-controller-amd64 /app/lfsx-web-controller-amd64

# Run as non-root
USER 1001

CMD [ "/app/lfsx-web-controller-amd64" ]

# Information label
LABEL maintainer="Hama GmbH <Jonas.Letzbor@hama.de>" \
    org.label-schema.description="Controller to enable running a LFS.X inside your browser"

# Image bauen:     buildah bud --layers --network host --secret id=giteaSshKey,src=/home/ubuntugui/.secrets/gitea_ssh-key --tag=hama.de/lfsx-web-controller:v0
# Image ausführen: podman run hama.de/lfsx-web-controller:v0