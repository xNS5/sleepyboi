#!/bin/bash

curr_color_scheme=$(gsettings get org.gnome.desktop.interface color-scheme)
curr_gtk_theme=$(gsettings get org.gnome.desktop.interface gtk-theme)

if [ "$curr_color_scheme" != "prefer-dark" ]; then
    gsettings set org.gnome.desktop.interface color-scheme 'prefer-dark'
fi

if [ "$curr_gtk_theme" != "Pop-dark" ]; then
    gsettings set org.gnome.desktop.interface gtk-theme 'Pop-dark'
fi

rm /var/lock/sleepyboi.lock 2>/dev/null 
exit 0