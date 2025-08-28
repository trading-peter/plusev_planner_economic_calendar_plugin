package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/plusev-terminal/go-plugin-common/logging"
	pi "github.com/plusev-terminal/go-plugin-common/planner/import"
	"github.com/plusev-terminal/go-plugin-common/requester"

	"github.com/extism/go-pdk"
)

//go:wasmexport import_events
func import_events() int32 {
	// Initialize logger
	logger := logging.NewLogger("example-plugin")

	job := pi.ImportJob{}
	err := pdk.InputJSON(&job)
	if err != nil {
		logger.ErrorWithData("Failed to parse input JSON", map[string]any{
			"error": err.Error(),
		})
		return 1
	}

	fromStr := job.From.Format("2006-01-02T15:04:05")
	toStr := job.To.Format("2006-01-02T15:04:05")

	queryStr := fmt.Sprintf("date_mode=1&from=%s&to=%s&importance=15&currencies=262143", url.QueryEscape(fromStr), url.QueryEscape(toStr))

	req := requester.Request{
		Method: "POST",
		URL:    "https://www.mql5.com/en/economic-calendar/content",
		Body:   []byte(queryStr),
		Headers: map[string]string{
			"accept":           "*/*",
			"accept-language":  "en-US,en;q=0.9,de-DE;q=0.8,de;q=0.7,en-DE;q=0.6",
			"cache-control":    "no-cache",
			"content-type":     "application/x-www-form-urlencoded",
			"user-agent":       "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36",
			"x-requested-with": "XMLHttpRequest",
		},
	}

	resp, err := requester.Send(&req, nil)
	if err != nil {
		// Log the error with details
		logger.ErrorWithData("Failed to send HTTP request", map[string]any{
			"error": err.Error(),
			"url":   req.URL,
		})
		pdk.SetError(fmt.Errorf("failed to send request: %v", err))
		return 1
	}

	bodyText := resp.Body

	rawEvents := []map[string]any{}
	err = json.Unmarshal(bodyText, &rawEvents)
	if err != nil {
		logger.ErrorWithData("Failed to unmarshal response body", map[string]any{
			"error": err.Error(),
		})
		return 1
	}

	events := []pi.ImportEvent{}

	for _, e := range rawEvents {
		eventName := getValue[string]("EventName", e)
		if eventName == "" {
			continue
		}

		eventType := getValue[float64]("EventType", e, 1)
		currencyCode := getValue[string]("CurrencyCode", e)
		startDate := time.UnixMilli(int64(e["ReleaseDate"].(float64))).UTC()
		endDate := time.Time{}

		if eventType == 1 {
			endDate = startDate
		}

		actualValue := getValue("ActualValue", e, "N/A")
		previousValue := getValue("PreviousValue", e, "N/A")
		forecastValue := getValue("ForecastValue", e, "N/A")

		var notes string
		if anyMatches(func(v string) bool { return v != "N/A" }, actualValue, previousValue, forecastValue) {
			notes = fmt.Sprintf("Actual: %s | Forecast: %s | Previous: %s", actualValue, forecastValue, previousValue)
		}

		events = append(events, pi.ImportEvent{
			Title:     fmt.Sprintf("%s%s", ifThen(currencyCode != "", currencyCode+": ", ""), eventName),
			StartDate: startDate,
			EndDate:   endDate,
			Timezone:  "UTC",
			Notes:     notes,
		})
	}

	// Allocate memory and marshal to JSON
	mem, err := pdk.AllocateJSON(pi.ImportData{
		Events: events,
	})
	if err != nil {
		pdk.SetError(fmt.Errorf("failed to allocate memory for request: %v", err))
		return 1
	}

	ptr := calendarImport(mem.Offset())
	resultMem := pdk.FindMemory(ptr)
	respData := resultMem.ReadBytes()

	var res pi.ImportResult
	if err := json.Unmarshal(respData, &res); err != nil {
		logger.ErrorWithData("Failed to unmarshal import result", map[string]any{
			"error": err.Error(),
		})
		return 1
	}

	return 0
}

type mapValue interface {
	float64 | string | bool
}

func ifThen[T any](condition bool, trueValue T, falseValue T) T {
	if condition {
		return trueValue
	}
	return falseValue
}

func getValue[T mapValue](key string, data map[string]any, defaultValue ...T) T {
	value, ok := data[key].(T)
	if !ok {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		var defaultValue T
		return defaultValue
	}

	var zero T

	if len(defaultValue) > 0 && value == zero {
		return defaultValue[0]
	}

	return value
}

func anyMatches[T comparable](predicate func(T) bool, values ...T) bool {
	for _, v := range values {
		if predicate(v) {
			return true
		}
	}
	return false
}
