# sleepyboi

This is a systemd service for changing the system theme in accordance with sunset and sunrise written in Golang. 

This is designed specifically for Gnome-based desktop environments, but can be tweaked to work with other DE's.

Note: This is dependent on the [at](https://phoenixnap.com/kb/linux-at-command) command. 

## Why Golang?

Honestly? Fun. This script can easily be written in Python, Javascript, Java, Rust, Bash, you name it. The logic is pretty simple:

1. Get the general geographic location
2. Use that geographic information to get sunrise/sunset times
3. Run the respective sunrise/sunset script based on whether the current time is before or after sunset/sunrise. 
4. ???
5. Profit