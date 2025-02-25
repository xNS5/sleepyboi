#!/bin/bash

curr_color_scheme=$(gsettings get org.gnome.desktop.interface color-scheme)
curr_gtk_theme=$(gsettings get org.gnome.desktop.interface gtk-theme)

if [ "$curr_color_scheme" != "default" ]; then
    gsettings set org.gnome.desktop.interface color-scheme 'default'
fi

if [ "$curr_gtk_theme" != "Pop" ]; then
    gsettings set org.gnome.desktop.interface gtk-theme 'Pop'
fi

rm /var/lock/sleepyboi.lock 2>/dev/null
exit 0