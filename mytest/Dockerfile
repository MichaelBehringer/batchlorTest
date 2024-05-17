# Verwenden Sie ein Basis-Image mit Ubuntu
FROM ubuntu:latest

# Umgehen Sie Dialoge während der Installation
ENV DEBIAN_FRONTEND=noninteractive

# Installieren Sie notwendige Pakete
RUN apt-get update && apt-get install -y \
    wget \
    openjdk-11-jdk \
    tightvncserver \
    xfce4 \
    xfce4-goodies

# Laden und installieren Sie die Eclipse IDE
RUN wget http://download.eclipse.org/technology/epp/downloads/release/2020-03/R/eclipse-java-2020-03-R-linux-gtk-x86_64.tar.gz -O /tmp/eclipse.tar.gz && \
    tar -xzf /tmp/eclipse.tar.gz -C /opt && \
    rm /tmp/eclipse.tar.gz

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
