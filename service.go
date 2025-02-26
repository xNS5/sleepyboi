package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

func MakeRequest(url string) (map[string]any, error) {
	httpClient := http.Client{}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var responseJson map[string]any

	json.Unmarshal([]byte(string(body)), &responseJson)

	return responseJson, nil
}

func parseTime(date, timeStr, location string) (time.Time, error) {

	layout := "2006-01-02 3:04:05 PM"
	dateTimeStr := fmt.Sprintf("%s %s", date, timeStr)

	loc, err := time.LoadLocation(location)
	if err != nil {
		return time.Time{}, err
	}

	t, err := time.ParseInLocation(layout, dateTimeStr, loc)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func execNow(args []string) (response string, error error) {

	output, err := exec.Command(args[0], args[1:]...).Output()

	if err != nil {
		log.Fatalf("Error running %s: %v\r\n", strings.Join(args, " "), err)
		return "", err
	} else if output != nil {
		log.Printf("Running %s success\r\n", strings.Join(args, " "))
		return string(output), nil
	}

	return "", nil
}

func main() {
	iana_response, iana_err := MakeRequest("http://ip-api.com/json/")
	get_command := []string{"gsettings", "get"}
	set_command := []string{"gsettings", "set"}
	curr_color_scheme_cmd := []string{"org.gnome.desktop.interface", "color-scheme"}
	curr_gtk_theme_cmd := []string{"org.gnome.desktop.interface", "gtk-theme"}

	curr_time := time.Now()

	if iana_err != nil {
		log.Fatalf("Error getting remote iana name: %v", iana_err)
		return
	}

	latitude := iana_response["lat"].(float64)
	longitude := iana_response["lon"].(float64)
	url := fmt.Sprintf("https://api.sunrisesunset.io/json?lat=%f&lng=%f", latitude, longitude)

	sunrise_sunset_response, sunrise_sunset_err := MakeRequest(url)

	if sunrise_sunset_err != nil || sunrise_sunset_response["status"].(string) != "OK" {
		log.Fatalf("Error getting sunset/sunrise info: %v", sunrise_sunset_err)
		return
	}

	result := sunrise_sunset_response["results"].(map[string]any)
	date := result["date"].(string)
	timezone := result["timezone"].(string)
	sunrise := result["sunrise"].(string)
	sunset := result["sunset"].(string)

	sunrise_time, sunrise_err := parseTime(date, sunrise, timezone)

	if sunrise_err != nil {
		log.Fatalf("Error parsing sunrise time string: %v", sunrise_err)
		return
	}

	sunset_time, sunset_err := parseTime(date, sunset, timezone)

	if sunset_err != nil {
		log.Fatalf("Error parsing sunset time string: %v", sunset_err)
		return
	}

	log.Println(fmt.Sprintf("Current: %s\nSunrise: %s\nSunset: %s", curr_time.String(), sunrise_time.String(), sunset_time.String()))

	curr_color_scheme, color_scheme_err := execNow(append(get_command, curr_color_scheme_cmd...))

	if color_scheme_err != nil {
		log.Fatalf("Error getting current color scheme: %v", color_scheme_err)
		return
	}

	curr_gtk_theme, gtk_scheme_err := execNow(append(get_command, curr_gtk_theme_cmd...))

	if gtk_scheme_err != nil {
		log.Fatalf("Error getting current color scheme: %v", gtk_scheme_err)
	}

	if curr_time.Before(sunrise_time) {
		if curr_color_scheme != "prefer-dark" {
			_, err := execNow(append(set_command, append(curr_color_scheme_cmd, "\"prefer-dark\"")...))
			if err != nil {
				log.Fatalf("Error setting color scheme: %v", err)
				return
			}
		}
		if curr_gtk_theme != "Pop-dark" {
			_, err := execNow(append(set_command, append(curr_gtk_theme_cmd, "\"Pop-dark\"")...))
			if err != nil {
				log.Fatalf("Error setting GTK color theme: %v", err)
				return
			}
		}
	} else if curr_time.After(sunrise_time) && curr_time.Before(sunset_time) {
		if curr_color_scheme != "default" {
			_, err := execNow(append(set_command, append(curr_color_scheme_cmd, "\"default\"")...))
			if err != nil {
				log.Fatalf("Error setting color scheme: %v", err)
				return
			}
		}
		if curr_gtk_theme != "default" {
			_, err := execNow(append(set_command, append(curr_gtk_theme_cmd, "\"Pop\"")...))
			if err != nil {
				log.Fatalf("Error setting GTK color theme: %v", err)
				return
			}
		}
	}
}
