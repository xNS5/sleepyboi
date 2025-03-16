# sleepyboi

This is a systemd service for changing the system theme in accordance with sunset and sunrise written in Golang. 

This is designed specifically for Gnome-based desktop environments, but can be tweaked to work with other DE's.

This service is designed to run once a minute, leverageing a `systemd` timer.

## Why Golang?

Honestly? Fun. This script can easily be written in Python, Javascript, Java, Rust, Bash, you name it. The logic is pretty simple:

1. Get the general geographic location
2. Use that geographic information to get sunrise/sunset times
3. If the current time is after sunrise, execute sunrise theme command
4. If the current time is after susnet, execute sunset theme command  
5. ???
6. Profit


This service is powered by [SunriseSunset.io](https://sunrisesunset.io/api/)