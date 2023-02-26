package addtocalendars

import (
	"errors"
	"net/url"
	"time"

	"github.com/inayatmemon/addtocalendar/local_time"
)

type AddToCalendar struct {
	Title                   string
	Details                 string
	Location                string
	Timezone                string
	EventStartUnixTimestamp int64
	EventEndUnixTimestamp   int64
}

func (addToCalendar *AddToCalendar) AddToCalendar() (string, error) {
	var emptyStruct AddToCalendar
	if *addToCalendar == emptyStruct {
		return "", errors.New("Please provide atleast one value to struct")
	}

	startTime := ""
	endTime := ""
	var locTime *time.Location
	locTime, _ = time.LoadLocation("")

	calendarUrl := "https://calendar.google.com/calendar/render?action=TEMPLATE"

	if addToCalendar.Title != "" {
		addToCalendar.Title = url.QueryEscape(addToCalendar.Title)
		calendarUrl += "&text=" + addToCalendar.Title
	}

	if addToCalendar.Details != "" {
		addToCalendar.Details = url.QueryEscape(addToCalendar.Details)
		calendarUrl += "&details=" + addToCalendar.Details
	}

	if addToCalendar.Location != "" {
		addToCalendar.Location = url.QueryEscape(addToCalendar.Location)
		calendarUrl += "&location=" + addToCalendar.Location
	}

	if addToCalendar.Timezone != "" {

		data, err := readLocationData(addToCalendar.Timezone)
		if err != nil {
			return "", errors.New("Invalid timezone data")
		}

		locTime, err = time.LoadLocationFromTZData(addToCalendar.Timezone, data)
		if err != nil {
			locTime = time.Local
		}

		addToCalendar.Timezone = url.QueryEscape(addToCalendar.Timezone)
		calendarUrl += "&ctz=" + addToCalendar.Timezone
	}

	if addToCalendar.EventStartUnixTimestamp != 0 {
		startTime = time.Unix(addToCalendar.EventStartUnixTimestamp, 0).Format("20060102T150405")
		if addToCalendar.Timezone != "" {
			startTime = time.Unix(addToCalendar.EventStartUnixTimestamp, 0).In(locTime).Format("20060102T150405")
		}
		endTime = startTime
	}

	if addToCalendar.EventEndUnixTimestamp != 0 {
		endTime = time.Unix(addToCalendar.EventEndUnixTimestamp, 0).Format("20060102T150405")
		if addToCalendar.Timezone != "" {
			endTime = time.Unix(addToCalendar.EventEndUnixTimestamp, 0).In(locTime).Format("20060102T150405")
		}
	}

	if startTime != "" && endTime != "" {
		calendarUrl += "&dates=" + startTime + "/" + endTime
	} else if startTime != "" && endTime == "" {
		calendarUrl += "&dates=" + startTime + "/" + startTime
	} else if startTime == "" && endTime != "" {
		calendarUrl += "&dates=" + endTime + "/" + endTime
	}

	return calendarUrl, nil
}

func readLocationData(timezone string) ([]byte, error) {

	zipFile := "zoneinfo.zip"

	databytes, err2 := local_time.LoadTzinfoFromZip(zipFile, timezone)
	if err2 != nil {
		return nil, err2
	}

	return databytes, nil
}
