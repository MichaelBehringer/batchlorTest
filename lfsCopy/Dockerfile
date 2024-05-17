ARG ALPINE_VERSION=3.19

FROM alpine:${ALPINE_VERSION} AS guacamole-builder

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

ARG BUILD_DIR=/tmp/guacamole-server
COPY guacamole-server/ ${BUILD_DIR}
RUN chmod -R 777 ${BUILD_DIR}

ARG PREFIX_DIR=/opt/guacamole

ARG WITH_FREERDP="NO"
ARG WITH_LIBSSH2='libssh2-\d+(\.\d+)+'
ARG WITH_LIBTELNET='\d+(\.\d+)+'
ARG WITH_LIBVNCCLIENT='LibVNCServer-\d+(\.\d+)+'
ARG WITH_LIBWEBSOCKETS='v\d+(\.\d+)+'

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

RUN ${BUILD_DIR}/src/guacd-docker/bin/build-all.sh

RUN ${BUILD_DIR}/src/guacd-docker/bin/list-dependencies.sh \
        ${PREFIX_DIR}/sbin/guacd               \
        ${PREFIX_DIR}/lib/libguac-client-*.so  \
        > ${PREFIX_DIR}/DEPENDENCIES

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

RUN apk add -X https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VERSION}/main -u alpine-keys --allow-untrusted
RUN apk update && apk upgrade

RUN apk add --no-cache socat sway xkeyboard-config wayvnc foot bash \
    openjdk11-jre gtk+3.0 python3 gcompat gsettings-desktop-schemas \
    py3-numpy py3-pip libcap nano curl mesa-dri-gallium gtk-update-icon-cache  \
    ${DEV_DEPENDENCIES}

ARG PREFIX_DIR_GUACAMOLE=/opt/guacamole

COPY --from=guacamole-builder ${PREFIX_DIR_GUACAMOLE} ${PREFIX_DIR_GUACAMOLE}

RUN xargs apk add --no-cache < ${PREFIX_DIR_GUACAMOLE}/DEPENDENCIES

EXPOSE 4822

RUN python3 -m venv ~/pyvenv --system-site-packages \
    && PIP_CERT=/etc/ssl/certs/ca-certificates.crt ~/pyvenv/bin/pip3 install vncdotool

RUN addgroup -S ${USER} -g 1001 && adduser -S ${USER} -G ${USER} --uid 1001 -s /bin/ash

RUN mkdir -p /.config/glib-2.0/ && chown -R oracle:oracle /.config/glib-2.0/ && chmod -R 0777 /.config/glib-2.0/

RUN mv /etc/sway/config /etc/sway/config_original
COPY configs/config_sway /etc/sway/config

RUN mkdir /etc/sway/config.d \
    && printf "\
        exec 'socat TCP-LISTEN:7023,fork UNIX-CONNECT:/tmp/sway-ipc.sock' \n\
        exec 'wayvnc --keyboard=de --render-cursor 0.0.0.0 5910'" \
    > /etc/sway/config.d/exec

RUN apk del ${DEV_DEPENDENCIES}

EXPOSE 5910

RUN mkdir -p /usr/share/themes/Adwaita-Own/gtk-3.0/
COPY configs/gtk.css /usr/share/themes/Adwaita-Own/gtk-3.0/gtk.css

RUN apk add firefox
RUN apk add --no-cache ttf-dejavu ttf-freefont ttf-liberation
# COPY mozilla.cfg /usr/lib/firefox/mozilla.cfg
# RUN echo -e 'pref("gfx.xrender.enabled", false);\npref("gfx.canvas.accelerated", false);\npref("layers.acceleration.disabled", true);\npref("security.sandbox.content.level", 0);' > /usr/lib/firefox/defaults/pref/autoconfig.js

RUN echo -e 'pref("gfx.xrender.enabled", false);\r\npref("gfx.canvas.accelerated", false);\r\npref("layers.acceleration.disabled", true);' > /usr/lib/firefox/defaults/pref/autoconfig.js

ENV MOZ_ENABLE_WAYLAND=1 \
    MOZ_ACCELERATED=0 \
    MOZ_WEBRENDER=0

RUN apk add wget
RUN wget https://www.mirrorservice.org/sites/download.eclipse.org/eclipseMirror/technology/epp/downloads/release/2024-03/R/eclipse-jee-2024-03-R-linux-gtk-aarch64.tar.gz -O /tmp/eclipse.tar.gz && \
    tar -xzf /tmp/eclipse.tar.gz -C /opt && \
    rm /tmp/eclipse.tar.gz

USER 1001

RUN mkdir -p /home/${USER}/.config/wayvnc \
    && printf "\
        enable_auth=false \
    " > /home/${USER}/.config/wayvnc/config

RUN gsettings set org.gnome.desktop.interface icon-theme 'Adwaita'  \
    && gsettings set org.gnome.desktop.interface font-name 'OpenSans-Light 8.5' \
    && gsettings set org.gnome.desktop.interface monospace-font-name 'Source Code Pro 9' \
    && gsettings set org.gnome.desktop.interface document-font-name 'Arial 9' \
    && gsettings set org.gnome.desktop.interface gtk-enable-primary-paste false \
    && gsettings set org.gnome.desktop.interface font-antialiasing 'rgba' \
    && gsettings set org.gnome.desktop.interface gtk-theme 'Adwaita-Own'

USER 0

RUN setcap -r /usr/bin/sway

COPY --chmod=555 entrypoint.sh /entrypoint.sh

USER 0

RUN chown -R ${USER}:${USER} /home/${USER} \
    && chmod -R g+w /home/${USER}/ \
    && chmod -R g+r /home/${USER}/ \
    && chmod -R g+X /home/${USER}/

RUN mkdir /c && chown oracle:oracle /c

USER 1001

ENTRYPOINT ["/entrypoint.sh"]

LABEL maintainer="Der Coolere Jonas" \
    org.label-schema.description="Container for not running LFS.X inside your browser"