ARG ALPINE_VERSION=3.19

# Build Apache Guacamole. See this Dockerfile for more informations: https://github.com/apache/guacamole-server/blob/master/Dockerfile
FROM alpine:${ALPINE_VERSION} AS guacamole-builder

# Add certificates
COPY lfs/docker/certs/*.pem /etc/ssl/certs/ 
# First we need to be able to add certificates
RUN cat /etc/ssl/certs/hama-root.pem >> /etc/ssl/certs/ca-certificates.crt \
    && apk --no-cache add ca-certificates \
    && update-ca-certificates

# Install build dependencies
RUN apk add --no-cache                \
        autoconf                      \
        automake                      \
        build-base                    \
        cairo-dev                     \
        cmake                         \
        cunit-dev                     \
        git                           \
        grep                          \
        libjpeg-turbo-dev             \
        libpng-dev                    \
        libtool                       \
        libwebp-dev                   \
        make                          \
        openssl-dev                   \
        pango-dev                     \
        pulseaudio-dev                \
        util-linux-dev                \
        ffmpeg-libavformat            \
        ffmpeg-libavutil              \
        libvorbis-dev


# Copy source to container for sake of build
ARG BUILD_DIR=/tmp/guacamole-server
COPY guacamole-server/ ${BUILD_DIR}

#
# Base directory for installed build artifacts.
#
# NOTE: Due to limitations of the Docker image build process, this value is
# duplicated in an ARG in the second stage of the build.
#
ARG PREFIX_DIR=/opt/guacamole

#
# Automatically select the latest versions of each core protocol support
# library (these can be overridden at build time if a specific version is
# needed)#
# A regex for Git TAGS is used
#
ARG WITH_FREERDP="NO"
ARG WITH_LIBSSH2='libssh2-\d+(\.\d+)+'
ARG WITH_LIBTELNET='\d+(\.\d+)+'
ARG WITH_LIBVNCCLIENT='LibVNCServer-\d+(\.\d+)+'
ARG WITH_LIBWEBSOCKETS='v\d+(\.\d+)+'

#
# Default build options for each core protocol support library, as well as
# guacamole-server itself (these can be overridden at build time if different
# options are needed)
#

ARG GUACAMOLE_SERVER_OPTS="\
    --disable-guaclog"

ARG LIBSSH2_OPTS="\
    -DBUILD_EXAMPLES=OFF \
    -DBUILD_SHARED_LIBS=ON"

ARG LIBTELNET_OPTS="\
    --disable-static \
    --disable-util"

ARG LIBVNCCLIENT_OPTS=""

ARG LIBWEBSOCKETS_OPTS="\
    -DDISABLE_WERROR=ON \
    -DLWS_WITHOUT_SERVER=ON \
    -DLWS_WITHOUT_TESTAPPS=ON \
    -DLWS_WITHOUT_TEST_CLIENT=ON \
    -DLWS_WITHOUT_TEST_PING=ON \
    -DLWS_WITHOUT_TEST_SERVER=ON \
    -DLWS_WITHOUT_TEST_SERVER_EXTPOLL=ON \
    -DLWS_WITH_STATIC=OFF"

# Build guacamole-server and its core protocol library dependencies
RUN ${BUILD_DIR}/src/guacd-docker/bin/build-all.sh

# Record the packages of all runtime library dependencies
RUN ${BUILD_DIR}/src/guacd-docker/bin/list-dependencies.sh \
        ${PREFIX_DIR}/sbin/guacd               \
        ${PREFIX_DIR}/lib/libguac-client-*.so  \
        > ${PREFIX_DIR}/DEPENDENCIES



# Container for bulding the LFS.X desktop client
FROM alpine:${ALPINE_VERSION} AS builder

ARG ALPINE_VERSION \
    LFS_RELEASE \
    LFS_RELEASE_BRANCH

# Add certificates
COPY lfs/docker/certs/*.pem /etc/ssl/certs/ 
# First we need to be able to add certificates
RUN cat /etc/ssl/certs/hama-root.pem >> /etc/ssl/certs/ca-certificates.crt \
    && apk --no-cache add ca-certificates \
    && update-ca-certificates

# Update system
RUN apk add -X https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VERSION}/main -u alpine-keys --allow-untrusted \
    && apk update && apk upgrade

# Install build dependencies
RUN apk add --no-cache git maven openjdk11-jre curl

# Add cacerts for java
COPY lfs/docker/certs/cacerts-11 /usr/lib/jvm/java-11-openjdk/lib/security/cacerts

# Argument to disable cache if repository was changed
ARG lfsLastCommitHash
# Build Java LFS
COPY lfs/docker/configs/features.patch /build/features.patch
RUN --mount=type=secret,id=giteaApiKey,required=true mkdir -p /build/.m2 \
    && cd /build \
    && git clone --depth 1 --branch ${LFS_RELEASE_BRANCH} "https://notNeeded:"$(cat /run/secrets/giteaApiKey)"@gitea.hama.de/LFS/hama.java.lfs.git" \
    && cd /build/hama.java.lfs \
    && git apply --ignore-space-change --ignore-whitespace --reject /build/features.patch
 # Git needed for OSGi bundle -> can't download zip file
 #   && curl -X GET https://gitea.hama.de/api/v1/repos/LFS/hama.java.lfs/archive/${LFS_RELEASE_BRANCH}.zip -H "Authorization: token "$(cat /run/secrets/giteaApiKey)"" -o lfs.zip \
 #   && unzip lfs.zip

COPY lfs/docker/configs/config_maven /build/settings.xml

RUN cd /build \
    && JAVA_HOME=/usr/lib/jvm/java-11-openjdk mvn org.eclipse.tycho:tycho-versions-plugin:set-version -f hama.java.lfs/de.hama.lfs.maven/pom.xml -Duser.home=/build -DnewVersion=${LFS_RELEASE} -s /build/settings.xml \
    && JAVA_HOME=/usr/lib/jvm/java-11-openjdk mvn clean install -f hama.java.lfs/de.hama.lfs.maven/pom.xml -Duser.home=/build -s /build/settings.xml
 #   && mkdir /opt/lfsx/ \
 #   && cp -r hama.java.lfs/de.hama.lfs.repository/target/products/de.hama.lfs.platform.product/linux/gtk/x86_64/* /opt/lfsx/


# Build app
FROM docker.io/golang:1.20-alpine3.17 AS builderGo
ARG VERSION="0.0.0"

WORKDIR /build

# Add certificates
COPY lfs/docker/certs/* /etc/ssl/certs/ 
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

# Build
ENV GOINSECURE=proxy.golang.org
RUN --mount=type=secret,id=giteaSshKey,required=true cd ./lfs \
    && GOOS=linux GOARCH=amd64 go build -o "lfsx-web-lfs-amd64" -ldflags "-X main.version=${VERSION}}" ./cmd/lfsx-web-lfs


FROM alpine:${ALPINE_VERSION}
ARG ALPINE_VERSION

ENV USER="oracle" \
    DEV_DEPENDENCIES="wget openssl" \
    WLR_BACKENDS="headless" \
    WLR_LIBINPUT_NO_DEVICES="1" \
    WAYLAND_DISPLAY="wayland-1" \
    XDG_RUNTIME_DIR="/tmp" \
    XDG_SESSION_TYPE="wayland" \
    SWAYSOCK="/tmp/sway-ipc.100.2.sock" \
    HOME="/home/oracle/" \
    # Allow software rendering for OpenGL hardware acceleration
    WLR_RENDERER_ALLOW_SOFTWARE="1" \
    # Use software rendered cursor
    WLR_NO_HARDWARE_CURSORS="1" \
    # Number of virtual outputs for "headless" backend
    WLR_HEADLESS_OUTPUTS="1" \
    # Force the creaton of "hardware accelerated renderers" (vulkan, pixman)
    WRL_RENDERER="gles2" \
    LIBGL_ALWAYS_SOFTWARE=1 \
    # Use wayland for GTK 
    GDK_BACKEND=wayland

# Add certificates
COPY lfs/docker/certs/* /etc/ssl/certs/ 
# First we need to be able to add certificates
RUN cat /etc/ssl/certs/hama-root.pem >> /etc/ssl/certs/ca-certificates.crt \
    && apk --no-cache add ca-certificates \
    && update-ca-certificates

# Update system
RUN apk add -X https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VERSION}/main -u alpine-keys --allow-untrusted
RUN apk update && apk upgrade

# Add packages
RUN apk add --no-cache socat sway xkeyboard-config wayvnc foot bash \
    openjdk11-jre gtk+3.0 python3 gcompat gsettings-desktop-schemas \
    py3-numpy py3-pip libcap nano curl mesa-dri-gallium gtk-update-icon-cache  \
    ${DEV_DEPENDENCIES}

# Add fonts
RUN apk add --no-cache msttcorefonts-installer fontconfig ttf-opensans adwaita-icon-theme \
    && update-ms-fonts

# Copy guacamole
ARG PREFIX_DIR_GUACAMOLE=/opt/guacamole

# Copy build artifacts into this stage
COPY --from=guacamole-builder ${PREFIX_DIR_GUACAMOLE} ${PREFIX_DIR_GUACAMOLE}

# Install required dependencies
RUN xargs apk add --no-cache < ${PREFIX_DIR_GUACAMOLE}/DEPENDENCIES

# Expose the default listener port
EXPOSE 4822

# Install vncdotool
RUN python3 -m venv ~/pyvenv --system-site-packages \
    && PIP_CERT=/etc/ssl/certs/ca-certificates.crt ~/pyvenv/bin/pip3 install vncdotool

# Add user to run the application
RUN addgroup -S ${USER} -g 1001 && adduser -S ${USER} -G ${USER} --uid 1001 -s /bin/ash

# Required for GLib
RUN mkdir -p /.config/glib-2.0/ && chown -R oracle:oracle /.config/glib-2.0/ && chmod -R 0777 /.config/glib-2.0/

# Copy sway configuration
RUN mv /etc/sway/config /etc/sway/config_original
COPY lfs/docker/configs/config_sway /etc/sway/config

# Run programs on startup of sway
RUN mkdir /etc/sway/config.d \
    && printf "\
        exec 'socat TCP-LISTEN:7023,fork UNIX-CONNECT:/tmp/sway-ipc.sock' \n\
        exec 'wayvnc --keyboard=de --render-cursor 0.0.0.0 5910'" \
    > /etc/sway/config.d/exec

# Copy LFS and configuration
RUN mkdir /opt/lfsx \
    && mkdir -p /opt/lfs-user/config-prod && mkdir -p /opt/lfs-user/config-dev && chown -R ${USER}:${USER} /opt/lfs-user
COPY --from=builder /build/hama.java.lfs/de.hama.lfs.repository/target/products/de.hama.lfs.platform.product/linux/gtk/x86_64/. /opt/lfsx/
COPY lfs/docker/configs/lfsx-dev.properties /opt/lfs-user/config-dev/lfsx.properties
COPY lfs/docker/configs/lfsx-prod.properties /opt/lfs-user/config-prod/lfsx.properties

# Delete dev dependencies
RUN apk del ${DEV_DEPENDENCIES}

# Expose the VNC Port
EXPOSE 5910

# Copy custom GTK theme
RUN mkdir -p /usr/share/themes/Adwaita-Own/gtk-3.0/
COPY lfs/docker/configs/gtk.css /usr/share/themes/Adwaita-Own/gtk-3.0/gtk.css

# Temp: Add firefox
RUN apk add firefox
# Disable hardware acceleration. Profiles a random so we can't use user.js file. So we use autoconfig for firefox
RUN echo -e 'pref("gfx.xrender.enabled", false);\r\npref("gfx.canvas.accelerated", false);\r\npref("layers.acceleration.disabled", true);' > /usr/lib/firefox/defaults/pref/autoconfig.js

ENV MOZ_ENABLE_WAYLAND=1 \
    MOZ_ACCELERATED=0 \
    MOZ_WEBRENDER=0

# Change user to oracle
USER 1001

# Configure swayvnc
RUN mkdir -p /home/${USER}/.config/wayvnc \
    && printf "\
        enable_auth=false \
    " > /home/${USER}/.config/wayvnc/config

# Configure fonts
RUN gsettings set org.gnome.desktop.interface icon-theme 'Adwaita'  \
    && gsettings set org.gnome.desktop.interface font-name 'OpenSans-Light 8.5' \
    && gsettings set org.gnome.desktop.interface monospace-font-name 'Source Code Pro 9' \
    && gsettings set org.gnome.desktop.interface document-font-name 'Arial 9' \
    && gsettings set org.gnome.desktop.interface gtk-enable-primary-paste false \
    && gsettings set org.gnome.desktop.interface font-antialiasing 'rgba' \
    && gsettings set org.gnome.desktop.interface gtk-theme 'Adwaita-Own'

USER 0

# Sway tries by default to drop all capabilities. But we don't want to give sway the capability 'CAP_SYS_NICE' to be able to drop
# these rights. Because we limit the rights of the process already be deafult we don't need that feature
RUN setcap -r /usr/bin/sway

# Add Go application
RUN mkdir /opt/go-lfs
COPY --from=builderGo --chmod=555 /build/lfs/lfsx-web-lfs-amd64 /opt/go-lfs/go-lfs

# Add entrypoint
COPY --chmod=555 lfs/docker/entrypoint.sh /entrypoint.sh

USER 0
# Edit permissions that also the "oracle" group can write files to the needed files
RUN chown -R ${USER}:${USER} /home/${USER} \
    && chmod -R g+w /home/${USER}/ \
    && chmod -R g+r /home/${USER}/ \
    && chmod -R g+X /home/${USER}/ \
    && chmod -R g+w /opt/lfs-user/ \
    && chmod -R g+w /opt/lfsx/ \
    && chown -R oracle:oracle /opt/lfsx/
# The only "security" concern in the above command is setting the folder /opt/lfsx/ to oracle. Otherwise an error with
# "g_error_new_literal: assertion 'message != NULL' failed" will appear and the LFS.X won't start. But I really don't know why?

# Some functions of the LFS.X even have a "C:\\" folder hard coded -> create this one with user permissions
RUN mkdir /c && chown oracle:oracle /c

USER 1001

ENTRYPOINT ["/entrypoint.sh"]

# Information label
LABEL maintainer="Hama GmbH <Jonas.Letzbor@hama.de>" \
    org.label-schema.description="Container for running LFS.X inside your browser"
