package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Logger zerolog.Logger

var get_command = []string{"gsettings", "get"}
var set_command = []string{"gsettings", "set"}
	
var color_scheme_cmd = []string{"org.gnome.desktop.interface", "color-scheme"}
var gtk_theme_cmd = []string{"org.gnome.desktop.interface", "gtk-theme"}

const (
	PROD = iota
	DEBUG
)

const MODE = DEBUG


type State struct {
	LastRun time.Time
	Latitude float64
	Longitude float64
	Sunrise time.Time
	Sunset time.Time
}

type OutState struct {
	LastRun string
	Latitude float64
	Longitude float64
	Sunrise string
	Sunset string
}

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

func ParseTime(date, timeStr, location string) (*time.Time, error) {

	layout := "2006-01-02 15:04:05"
	dateTimeStr := fmt.Sprintf("%s %s", date, timeStr)

	loc, err := time.LoadLocation(location)
	if err != nil {
		return nil, err
	}

	t, err := time.ParseInLocation(layout, dateTimeStr, loc)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func ExecNow(args []string) (response string, error error) {

	output, err := exec.Command(args[0], args[1:]...).Output()

	if err != nil {
		Logger.Error().Err(err).Msgf("Error running %s: %v", strings.Join(args, " "), err)
		return "", err
	} else if output != nil {

		if MODE == DEBUG {
			Logger.Debug().Msgf("Running `%s` success", strings.Join(args, " "))
		}

		return string(output), nil
	}

	return "", nil
}

func WriteState(state *State) error {

	home, err := os.UserHomeDir()

	if err != nil {
		return err
	}

	stateDir := filepath.Join(home, ".local/lib/sleepyboi")
	stateFile := filepath.Join(stateDir, "sleepyboi.json")

	jsonBytes, err := json.MarshalIndent(&OutState{
		LastRun: state.LastRun.Format(time.RFC3339),
		Latitude: state.Latitude,
		Longitude: state.Longitude,
		Sunrise: state.Sunrise.Format(time.RFC3339),
		Sunset: state.Sunset.Format(time.RFC3339),
	}, "", " ")

	if err != nil {
		return err
	}

	err = os.WriteFile(stateFile, jsonBytes, 0644)

	return err
}

func GetState(time_basis time.Time) (*State, error){
	latitude, longitude := GetCoords()
	sunrise_ts, sunset_ts := GetSunriseSunset(latitude, longitude)

	return &State{
		LastRun: time_basis,
		Latitude: *latitude,
		Longitude: *longitude,
		Sunrise: *sunrise_ts,
		Sunset: *sunset_ts,
	}, nil
}

func SetNewState() (*State, error) {

	now := time.Now()
	year, month, day := now.Date()
	location := now.Location()

	state, err := GetState(time.Date(year, month, day, 0, 0, 0, 0, location))

	if err != nil {
		return nil, err
	}

	if err := WriteState(state); err != nil {
		return nil, err
	}

	return state, err

}

func GetLocalState() (*State, error) {
	home, err := os.UserHomeDir()

	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(home, ".local/lib/sleepyboi")
	stateFile := filepath.Join(stateDir, "sleepyboi.json")

	info, err :=  os.Stat(stateFile); 

	if err != nil {
		return nil, err
	}

	isEmpty := info.Size() <= 1

	var stateMap *State


	if isEmpty {

		if MODE == DEBUG {
			Logger.Debug().Msgf("State file has no contents. Fetching remote state info...")
		}

		state, err := SetNewState()

		if err != nil {
			return nil, err
		}

		stateMap = state
	} else {
		if MODE == DEBUG {
			Logger.Debug().Msgf("Fetching local state info...")
		}
		file, _ := os.ReadFile(stateFile)
	
		err = json.Unmarshal(file, &stateMap)

		if err != nil {
			return nil, err
		}
	}


	if time.Now().Local().After(stateMap.LastRun) {
		
		if MODE == DEBUG {
			Logger.Debug().Msgf("Local state invalid, fetching new state...")
		}

		stateMap, err = SetNewState()

		if err != nil {
			return nil, err
		}

	}

	return stateMap, nil	
}

func GetCoords() (*float64, *float64){
	iana_response, iana_err := MakeRequest("http://ip-api.com/json/")
	
	if iana_err != nil {
		Logger.Error().Err(iana_err).Msg("Error getting remote iana name")
		return nil, nil
	}

	latitude := iana_response["lat"].(float64)
	longitude := iana_response["lon"].(float64)

	if MODE == DEBUG {
		Logger.Debug().Msgf("Latitude: %v Longitude: %v", latitude, longitude)
	}

	return &latitude, &longitude
}

func GetSunriseSunset(latitude, longitude *float64) (*time.Time, *time.Time) {

	curr_time := time.Now().Local()

	url := fmt.Sprintf("https://api.sunrisesunset.io/json?lat=%f&lng=%f&time_format=24&date=%s", *latitude, *longitude, curr_time.Format("2006-01-02")) 

	sunrise_sunset_response, sunrise_sunset_err := MakeRequest(url)

	if sunrise_sunset_response["status"].(string) != "OK" {
		Logger.Error().Msgf("Error getting sunset/sunrise info: %v", sunrise_sunset_response["body"])
		return nil, nil
	}

	if sunrise_sunset_err != nil  {
		Logger.Error().Err(sunrise_sunset_err).Msgf("Error getting sunset/sunrise info")
		return nil, nil
	}

	result := sunrise_sunset_response["results"].(map[string]any)
	date := result["date"].(string)
	timezone := result["timezone"].(string)
	sunrise := result["sunrise"].(string)
	sunset := result["sunset"].(string)

	sunrise_time, sunrise_err := ParseTime(date, sunrise, timezone)

	if sunrise_err != nil {
		Logger.Error().Err(sunrise_err).Msg("Error parsing sunrise time string")
		return nil, nil
	}

	sunset_time, sunset_err := ParseTime(date, sunset, timezone)

	if sunset_err != nil {
		Logger.Error().Err(sunset_err).Msg("Error parsing sunset time string")
		return nil, nil
	}

	if MODE == DEBUG {
		Logger.Debug().Msgf("Sunrise: %s Sunset: %s", sunrise_time.String(), sunset_time.String())
	}

	return sunrise_time, sunset_time
}

func SetDarkTheme() (bool, error) {

	color_scheme, gtk_theme, err := GetSystemTheme()

	did_run := false

	if err != nil {
		return false, err
	}

	if *color_scheme != "prefer-dark" {
		_, err := ExecNow(append(set_command, append(color_scheme_cmd, "\"prefer-dark\"")...))
		if err != nil {
			Logger.Error().Err(err).Msg("Error setting color scheme")
			return false, err
		}
		if MODE == DEBUG {
			Logger.Debug().Msg("Setting color scheme to prefer-dark")
		}
		did_run = true
	}
	if *gtk_theme != "Pop-dark" {
		_, err := ExecNow(append(set_command, append(gtk_theme_cmd, "\"Pop-dark\"")...))
		if err != nil {
			Logger.Error().Err(err).Msg("Error setting GTK color theme")
			return false, err
		}
		if MODE == DEBUG {
			Logger.Debug().Msg("Setting gtk theme to pop-dark")
		}
		did_run = true
	}

	return did_run, nil
}

func SetLightTheme() (bool, error) {
	color_scheme, gtk_theme, err := GetSystemTheme()

	did_run := false

	if err != nil {
		return false, err
	}
	
	if *color_scheme != "default" {
		_, err := ExecNow(append(set_command, append(color_scheme_cmd, "\"default\"")...))
		if err != nil {
			Logger.Error().Err(err).Msg("Error setting color scheme")
			return false, err
		}
		
		if MODE == DEBUG {
			Logger.Debug().Msg("Setting gtk theme to default")
		}

		did_run = true
	}
	if *gtk_theme != "Pop" {
		_, err := ExecNow(append(set_command, append(gtk_theme_cmd, "\"Pop\"")...))
		if err != nil {
			Logger.Error().Err(err).Msg("Error setting GTK color theme")
			return false, err
		}

		if MODE == DEBUG {
			Logger.Debug().Msg("Setting gtk theme to pop")
		}

		did_run = true
	}

	return did_run, nil
}

func GetSystemTheme() (*string, *string, error) {
	color_scheme, err := ExecNow(append(get_command, color_scheme_cmd...))

	if err != nil {
		Logger.Error().Err(err).Msg("Error getting current color scheme")
		return nil, nil, err
	}

	color_scheme = strings.TrimSpace(color_scheme)
	color_scheme = color_scheme[1 : len(color_scheme)-1]

	gtk_theme, err := ExecNow(append(get_command, gtk_theme_cmd...))

	if err != nil {
		Logger.Err(err).Err(err).Msg("Error getting current color scheme")
		return nil, nil, err
	}

	gtk_theme = strings.TrimSpace(gtk_theme)
	gtk_theme = gtk_theme[1 : len(gtk_theme)-1]

	return &color_scheme, &gtk_theme, nil
}

func main() {

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zeroLogger := log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		FormatLevel: func(i any) string {
			return strings.ToUpper(fmt.Sprintf("[%s]", i))
		},
	}).Level(zerolog.TraceLevel).With().Timestamp().Logger()

	Logger = zeroLogger

	curr_time := time.Now().Local()

	curr_state, err := GetLocalState()

	if err != nil {
		Logger.Error().Err(err).Str("service", "main").Msg("Error getting current state")
	}

	if curr_time.After(curr_state.Sunset) {
		if MODE == DEBUG {
			Logger.Debug().Msg("Current time after sunset")
		}

		if did_run, err := SetDarkTheme(); err != nil {
			Logger.Error().Err(err).Str("service", "main").Msg("Error setting dark theme")
		} else {
			if MODE == DEBUG {
				Logger.Debug().Msgf("Did run: %v", did_run)
			}
		}
	}  else if curr_time.After(curr_state.Sunrise) && curr_time.Before(curr_state.Sunset) {
		if MODE == DEBUG {
			Logger.Debug().Msg("Current time after sunrise and before sunset ")
		}

		if did_run, err := SetLightTheme(); err != nil {
			Logger.Error().Err(err).Str("service", "main").Msg("Error setting dark theme")
		} else {
			if MODE == DEBUG {
				Logger.Debug().Msgf("Did run: %v", did_run)
			}
		}
	}

	
}
