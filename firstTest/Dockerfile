# Verwenden Sie ein Basis-Image mit Ubuntu
FROM ubuntu:latest

# Umgehen Sie Dialoge während der Installation
ENV DEBIAN_FRONTEND=noninteractive

# Installieren Sie notwendige Pakete
RUN apt-get update && apt-get install -y \
    wget \
    openjdk-17-jdk \
    tightvncserver \
    xfce4 \
    xfce4-goodies \
    dbus-x11

# Laden und installieren Sie die Eclipse IDE
RUN wget https://www.mirrorservice.org/sites/download.eclipse.org/eclipseMirror/technology/epp/downloads/release/2024-03/R/eclipse-jee-2024-03-R-linux-gtk-aarch64.tar.gz -O /tmp/eclipse.tar.gz && \
    tar -xzf /tmp/eclipse.tar.gz -C /opt

# Setzen Sie ein Passwort für den VNC-Server
RUN mkdir /root/.vnc
RUN echo "test" | vncpasswd -f > /root/.vnc/passwd
RUN chmod 600 /root/.vnc/passwd

# Konfigurieren Sie die Startskripte für den VNC-Server
COPY startvnc.sh /opt/startvnc.sh
RUN chmod +x /opt/startvnc.sh

# Exponieren Sie den VNC-Port
EXPOSE 5901

# Starten Sie den VNC-Server, wenn der Container gestartet wird
CMD ["/opt/startvnc.sh"]
