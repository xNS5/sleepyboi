package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type SunTimes struct {
	Date       string `json:"date"`
	Sunrise    string `json:"sunrise"`
	Sunset     string `json:"sunset"`
	FirstLight string `json:"first_light"`
	LastLight  string `json:"last_light"`
	Dawn       string `json:"dawn"`
	Dusk       string `json:"dusk"`
	SolarNoon  string `json:"solar_noon"`
	GoldenHour string `json:"golden_hour"`
	Timezone   string `json:"timezone"`
	UTCOffset  int    `json:"utc_offset"` // Offset in minutes
}

type Response struct {
	Results SunTimes `json:"results"`
	Status  string   `json:"status"`
}

func MakeRequest(url string) (map[string]interface{}, error) {
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

	var responseJson map[string]interface{}

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
	iana_response, iana_err := MakeRequest("http://ip-api.com/json/")

	curr_time := time.Now()

	if iana_err != nil {
		log.Fatalf("Error getting remote iana name: %v", iana_err)
		return
	}

	var latitude = iana_response["lat"].(string)
	var longitude = iana_response["lon"].(string)
	var url = fmt.Sprintf("https://api.sunrisesunset.io/json?lat=%s&lng=%s", latitude, longitude)

	sunrise_sunset_response, sunrise_sunset_err := MakeRequest(url)

	if sunrise_sunset_err != nil || sunrise_sunset_response["status"].(string) != "OK" {
		log.Fatalf("Error getting sunset/sunrise info: %v", sunrise_sunset_err)
		return
	}

	var sunrise = sunrise_sunset_response["results"].(map[string]interface{})["sunrise"].(string)
	var sunset = sunrise_sunset_response["results"].(map[string]interface{})["sunrise"].(string)

}
