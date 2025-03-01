# sleepyboi

This is a systemd service for changing the system theme in accordance with sunset and sunrise written in Golang. 

This is designed specifically for Gnome-based desktop environments, but can be tweaked to work with other DE's.

This service is designed to run once a minute, leverageing a `systemd` timer.

## Why Golang?

Honestly? Fun. This script can easily be written in Python, Javascript, Java, Rust, Bash, you name it. The logic is pretty simple:

1. Get the general geographic location
2. Use that geographic information to get sunrise/sunset times
3. Run the respective sunrise/sunset script based on whether the current time is before or after sunset/sunrise. 
4. ???
5. Profit

## Note
* The Sunrise/Sunset API is a little funky. After the current day's `sunset_time` has passed, the API immediately returns the next day's sunset and sunrise time. I get why, but this behavior results in needing to compare if the data from the API is the current day or the next day. If it's the next day, odds are the sunset has passed. This is a working theory, and is subject to change on a whim.

This service is powered by [SunriseSunset.io](https://sunrisesunset.io/api/)