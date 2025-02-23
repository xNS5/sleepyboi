package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

func main() {

	_, error := os.Stat("/tmp/sleepyboi.lock")

	if error == nil {
		log.Print("Sleepyboi has already been run -- skipping")
		return
	}

	// Check if .lock file exists in
	iana_response, iana_err := MakeRequest("http://ip-api.com/json/")

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

	sunrise_time_obj, sunrise_err := parseTime(date, sunrise, timezone)

	if sunrise_err != nil {
		log.Fatalf("Error parsing sunrise time string: %v", sunrise_err)
		return
	}

	sunset_time_obj, sunset_err := parseTime(date, sunset, timezone)

	if sunset_err != nil {
		log.Fatalf("Error parsing sunset time string: %v", sunset_err)
		return
	}

	println(fmt.Sprintf("Current: %s\nSunrise: %s\nSunset: %s", curr_time.String(), sunrise_time_obj.String(), sunset_time_obj.String()))

	msg := []byte("Hello, world!")

	os.WriteFile("/tmp/sleepyboi.lock", msg, 0400)

}
