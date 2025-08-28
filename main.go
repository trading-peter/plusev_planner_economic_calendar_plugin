package main

import (
	"time"

	m "github.com/plusev-terminal/go-plugin-common/meta"
	"github.com/plusev-terminal/go-plugin-common/planner"
	pi "github.com/plusev-terminal/go-plugin-common/planner"
	"github.com/plusev-terminal/go-plugin-common/plugin"
	"github.com/plusev-terminal/go-plugin-common/requester"
	"github.com/plusev-terminal/go-plugin-common/utils"
)

// ============================================================================
// Main - Register the plugin
// ============================================================================

func init() {
	// Create plugin instance
	p := &Mql5Plugin{}

	// Register the plugin - this generates all standard WASM exports automatically
	// IMPORTANT: Must be in init(), not main(), so it runs before WASM exports are called
	plugin.RegisterPlugin(p)
}

func main() {
	// Required for WASM, but can be empty
}

type Mql5Plugin struct {
	config *plugin.ConfigStore
	client *Client
}

// GetMeta returns the plugin metadata
func (p *Mql5Plugin) GetMeta() m.Meta {
	return m.Meta{
		PluginID:    "mql5-economic-calendar-import",
		Name:        "MQL5 Economic Calendar Import",
		AppID:       "plusev_planner",
		Category:    "Import",
		Description: "Imports economic calendar events from MQL5",
		Author:      "trading_peter",
		Version:     "v1.0.0",
		Repository:  "https://github.com/trading-peter/plusev_planner_economic_calendar_plugin",
		Tags:        []string{},
		Contacts:    []m.AuthorContact{},
		Resources: m.ResourceAccess{
			AllowedNetworkTargets: []m.NetworkTargetRule{{Pattern: "https://www.mql5.com/en/economic-calendar/content/*"}},
			FsWriteAccess:         nil,
		},
	}
}

func (p *Mql5Plugin) GetRateLimits() []plugin.RateLimit {
	return []plugin.RateLimit{
		{
			Command: planner.CMD_IMPORT_EVENTS,
			Scope:   []plugin.RateLimitScope{plugin.RateLimitScopeIP},
			RPS:     plugin.CalculateRPS(1, time.Minute),
			Burst:   1,
		},
	}
}

// GetConfigFields returns the configuration fields needed by this plugin
func (p *Mql5Plugin) GetConfigFields() []plugin.ConfigField {
	// Initialize client if needed to get config fields
	if p.client == nil {
		p.client = NewClient(requester.NewRequester())
	}
	return p.client.GetConfigFields()
}

// OnInit is called when the plugin is initialized with user configuration
func (p *Mql5Plugin) OnInit(config *plugin.ConfigStore) error {
	p.config = config

	// Create WooX client
	p.client = NewClient(requester.NewRequester())

	return nil
}

// OnShutdown is called when the plugin is being shut down
func (p *Mql5Plugin) OnShutdown() error {
	// Cleanup resources if needed
	return nil
}

// RegisterCommands registers all command handlers
func (p *Mql5Plugin) RegisterCommands(router *plugin.CommandRouter) {
	router.Register(planner.CMD_IMPORT_EVENTS, p.handleImportEvents)
}

func (p *Mql5Plugin) handleImportEvents(params map[string]any) plugin.Response {
	importParams := pi.ImportParams{}
	err := utils.MapToStruct(params, &importParams)
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	events, err := p.client.FetchEvents(importParams)
	if err != nil {
		return plugin.ErrorResponse(err)
	}

	return plugin.SuccessResponse(events, time.Hour)
}
