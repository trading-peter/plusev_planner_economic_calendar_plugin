package main

import (
	"github.com/extism/go-pdk"
	m "github.com/plusev-terminal/go-plugin-common/meta"
)

func main() {}

//go:wasmimport extism:host/user calendar_import
func calendarImport(uint64) uint64

//go:wasmexport meta
func meta() int32 {
	pdk.OutputJSON(m.Meta{
		PluginID:    "mql5-economic-calendar-import",
		Name:        "MQL5 Economic Calendar Import",
		AppID:       "plusev_planner",
		Category:    "Import",
		Description: "Imports economic calendar events from MQL5",
		Author:      "PlusEV",
		Version:     "1.0.0",
		Repository:  "https://github.com/trading-peter/plusev_planner_economic_calendar_plugin",
		Tags:        []string{},
		Contacts:    []m.AuthorContact{},
		Resources: m.ResourceAccess{
			AllowedNetworkTargets: []m.NetworkTargetRule{{Pattern: "https://www.mql5.com/en/economic-calendar/content/*"}},
			FsWriteAccess:         nil,
			StdoutAccess:          true,
			StderrAccess:          true,
		},
	})

	return 0
}
