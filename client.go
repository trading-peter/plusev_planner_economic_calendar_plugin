package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/plusev-terminal/go-plugin-common/logging"
	"github.com/plusev-terminal/go-plugin-common/planner"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	rt "github.com/plusev-terminal/go-plugin-common/requester/types"
	"github.com/plusev-terminal/go-plugin-common/utils"
)

type Client struct {
	requester rt.RequestDoer
	log       *logging.Logger
}

func NewClient(requester rt.RequestDoer) *Client {
	return &Client{
		requester: requester,
		log:       logging.NewLogger("mql5-economic-calendar-import"),
	}
}

func (c *Client) FetchEvents(job planner.ImportParams) ([]planner.ImportEvent, error) {
	fromStr := job.From.Format("2006-01-02T15:04:05")
	toStr := job.To.Format("2006-01-02T15:04:05")

	queryStr := fmt.Sprintf("date_mode=1&from=%s&to=%s&importance=15&currencies=262143", url.QueryEscape(fromStr), url.QueryEscape(toStr))

	req := rt.Request{
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

	resp, err := c.requester.Send(&req, nil)
	if err != nil {
		// Log the error with details
		c.log.ErrorWithData("Failed to send HTTP request", map[string]any{
			"error": err.Error(),
			"url":   req.URL,
		})
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	bodyText := resp.Body

	rawEvents := []map[string]any{}
	err = json.Unmarshal(bodyText, &rawEvents)
	if err != nil {
		c.log.ErrorWithData("Failed to unmarshal response body", map[string]any{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	events := []planner.ImportEvent{}

	for _, e := range rawEvents {
		eventName := utils.GetValue[string]("EventName", e)
		if eventName == "" {
			continue
		}

		eventType := utils.GetValue[float64]("EventType", e, 1)
		currencyCode := utils.GetValue[string]("CurrencyCode", e)
		startDate := time.UnixMilli(int64(e["ReleaseDate"].(float64))).UTC()
		endDate := time.Time{}

		if eventType == 1 {
			endDate = startDate
		}

		actualValue := utils.GetValue("ActualValue", e, "N/A")
		previousValue := utils.GetValue("PreviousValue", e, "N/A")
		forecastValue := utils.GetValue("ForecastValue", e, "N/A")

		var notes string
		if utils.AnyMatches(func(v string) bool { return v != "N/A" }, actualValue, previousValue, forecastValue) {
			notes = fmt.Sprintf("Actual: %s | Forecast: %s | Previous: %s", actualValue, forecastValue, previousValue)
		}

		events = append(events, planner.ImportEvent{
			Title:     fmt.Sprintf("%s%s", utils.IfThen(currencyCode != "", currencyCode+": ", ""), eventName),
			StartDate: startDate,
			EndDate:   endDate,
			Timezone:  "UTC",
			Notes:     notes,
		})
	}

	return events, nil
}

func (c *Client) GetConfigFields() []plugin.ConfigField {
	// No config fields needed for this plugin
	return []plugin.ConfigField{}
}
