Da *swayvnc* noch keine lokalen (Software) Cursor in Kombination mit *Sway* unterstützt, behelfen wir uns einem kleinen Trick!  
Wir machen uns nämlich ein eigenes Cursor Theme, bei denen der Standard Cursor einfach unsichtbar ist.  

Problem ist aber immernoch, das wir Updates vom Cursor nicht erhalten.... Hierfür muss wayVNC angepasst werden.

Mit Sway könnten wir den Cursor zwar auch verstecken, nur hat der Client dann auch keinen mehr.

## Generieren

Für jeden Zeiger muss der folgenden Befehl ausgeführt werden, der das passende Image erstellt.

```sh
xcursorgen default.cursor default
```

## Anwenden

Um das Theme zu nutzen, muss der Ordner nach `/usr/share/icons` kopiert werden. Nach einigen Tests funktioniert der Benutzer Ordner `~/.local/share/icons/curstor/` oder `~/.icons` **NICHT**.

Anschließen kann das mit `gsettings` angewandt werden. Hinweis: wenn das Theme geändert wird, muss zunächst auf ein anderes Theme gewechselt werden!

```sh
gsettings set org.gnome.desktop.interface cursor-theme 'curstor'
gsettings set org.gnome.desktop.interface cursor-theme 'Yaru'
```

Damit das ganze auch effizient ist, sollten auch caches für die Icons erstellt werden.

```sh
update-icon-caches /usr/share/icons
```

