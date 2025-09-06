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
const (
	PROD = iota
	DEBUG
)

const MODE = DEBUG

var (
	STATE_FILE_NAME string
	CURR_TIME time.Time
	CURR_TIME_ZONE string
	STATE_FILE *State
	NEEDS_REFRESH=0
)

var (
	get_command = []string{"gsettings", "get"}
	set_command = []string{"gsettings", "set"}
	color_scheme_cmd = []string{"org.gnome.desktop.interface", "color-scheme"}
	gtk_theme_cmd = []string{"org.gnome.desktop.interface", "gtk-theme"}
	Logger zerolog.Logger
)

type Coordinates struct {
	TimeZone string
	Latitude float64
	Longitude float64
}

type State struct {
	LastRun time.Time
	Sunrise time.Time
	Sunset time.Time
	Timezone string
	Coordinates
}

type OutState struct {
	LastRun string
	Sunrise string
	Sunset string
	Timezone string
	Coordinates
}

func MakeRequest(url string) (map[string]any, error) {
	httpClient := http.Client{}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)

	if err != nil {
		NEEDS_REFRESH = 1
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
	jsonBytes, err := json.MarshalIndent(&OutState{
		LastRun: state.LastRun.Format(time.RFC3339),
		Sunrise: state.Sunrise.Format(time.RFC3339),
		Sunset: state.Sunset.Format(time.RFC3339),
		Coordinates: Coordinates{
			TimeZone: CURR_TIME_ZONE,
			Latitude: state.Latitude,
			Longitude: state.Longitude,
		},
	}, "", " ")

	if err != nil {
		return err
	}

	err = os.WriteFile(STATE_FILE_NAME, jsonBytes, 0644)

	return err
}

func GetState(time_basis time.Time) (*State, error){
	latitude, longitude := GetCoords()
	sunrise_ts, sunset_ts := GetSunriseSunset(latitude, longitude)

	return &State{
		LastRun: time_basis,
		Sunrise: *sunrise_ts,
		Sunset: *sunset_ts,
		Coordinates: Coordinates{
			TimeZone: CURR_TIME_ZONE,
			Latitude: *latitude,
			Longitude: *longitude,
		},
	}, nil
}

func SetNewState() (*State, error) {

	year, month, day := CURR_TIME.Date()
	location := CURR_TIME.Location()

	state, err := GetState(time.Date(year, month, day, 0, 0, 0, 0, location))

	if err != nil {
		return nil, err
	}

	if err := WriteState(state); err != nil {
		return nil, err
	}

	return state, err
}


func GetCoords() (*float64, *float64){
	
	if STATE_FILE.Coordinates.TimeZone == CURR_TIME_ZONE {
		return &STATE_FILE.Latitude, &STATE_FILE.Longitude
	}

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


func Init() error {

	// Time

	CURR_TIME = time.Now().Local()

	CURR_TIME_ZONE, _ = CURR_TIME.Zone()


	// Logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zeroLogger := log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		FormatLevel: func(i any) string {
			return strings.ToUpper(fmt.Sprintf("[%s]", i))
		},
	}).Level(zerolog.TraceLevel).With().Timestamp().Logger()

	Logger = zeroLogger

	// State
	home, err := os.UserHomeDir()

	if err != nil {
		return err
	}

	stateDir := filepath.Join(home, ".local/lib/sleepyboi")
	stateFile := filepath.Join(stateDir, "sleepyboi.json")

	STATE_FILE_NAME = stateFile

	if MODE == DEBUG {
		Logger.Debug().Msgf("Fetching local state info...")
	}

	info, err :=  os.Stat(STATE_FILE_NAME); 

	if err != nil {
		return err
	}

	isEmpty := info.Size() <= 1

	if isEmpty {

		if MODE == DEBUG {
			Logger.Debug().Msgf("State file has no contents. Fetching remote state info...")
		}

		state, err := SetNewState()

		if err != nil {
			return err
		}

		if MODE == DEBUG {
			Logger.Debug().Msg("Fetching remote state success")
		}

		STATE_FILE = state
	} else {

		file, _ := os.ReadFile(STATE_FILE_NAME)

		err = json.Unmarshal(file, &STATE_FILE)

		if err != nil {
			return err
		}

		if MODE == DEBUG {
			Logger.Debug().Msg("Fetching local state success")
		}
	}

	if CURR_TIME.After(STATE_FILE.LastRun) {
		
		if MODE == DEBUG {
			Logger.Debug().Msgf("Local state invalid, fetching new state...")
		}

		stateMap, err := SetNewState()

		if err != nil {
			return err
		}

		STATE_FILE = stateMap
	}

	return nil
}

func main() {

	if err := Init(); err != nil {
		Logger.Error().Err(err).Msg("Error initializing state")
		os.Exit(0)
	}

	if CURR_TIME.After(STATE_FILE.Sunset) {
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
	}  else if CURR_TIME.After(STATE_FILE.Sunrise) && CURR_TIME.Before(STATE_FILE.Sunset) {
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
