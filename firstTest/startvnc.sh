#!/bin/bash

# Setzen der Umgebungsvariablen für den Desktop
export USER=root
export DISPLAY=:1

# Bereinigen vorhandener VNC-Sessions
vncserver -kill :1 > /dev/null 2>&1 || true
rm -rf /tmp/.X1-lock /tmp/.X11-unix/X1 > /dev/null 2>&1 || true

# Initialisieren und starten des VNC-Servers
vncserver -kill :1
vncserver -geometry 1280x800 -depth 24 :1

# Starten der XFCE4 Desktop-Umgebung
startxfce4 &

# Warten, um sicherzustellen, dass der Desktop geladen wird
sleep 5

# Starten der Eclipse IDE
/opt/eclipse/eclipse &

# Halten des Containers am Laufen
while true; do sleep 1000; done
