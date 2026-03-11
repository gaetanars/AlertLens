package configbuilder

// ─── Time Intervals ──────────────────────────────────────────────────────────

// TimeIntervalEntry is the API/JSON model for a top-level named time interval
// block in an Alertmanager configuration.  It mirrors the YAML structure:
//
//	time_intervals:
//	  - name: business-hours
//	    time_intervals:
//	      - times: [{start_time: "09:00", end_time: "17:00"}]
//	        weekdays: ["monday:friday"]
//	        location: "Europe/Paris"
type TimeIntervalEntry struct {
	Name          string            `json:"name"           yaml:"name"`
	TimeIntervals []TimeIntervalDef `json:"time_intervals" yaml:"time_intervals"`
}

// TimeIntervalDef is one set of time conditions within a named time interval.
// All fields are optional — an empty def matches all times.
type TimeIntervalDef struct {
	// Times is a list of HH:MM time ranges.
	Times []TimeRangeDef `json:"times,omitempty" yaml:"times,omitempty"`
	// Weekdays accepts range strings such as "monday:friday" or "saturday".
	// Valid values: sunday, monday, tuesday, wednesday, thursday, friday, saturday.
	Weekdays []string `json:"weekdays,omitempty" yaml:"weekdays,omitempty"`
	// DaysOfMonth accepts range strings such as "1:15" or "-1" (last day).
	DaysOfMonth []string `json:"days_of_month,omitempty" yaml:"days_of_month,omitempty"`
	// Months accepts range strings such as "january:march" or "6" (numeric).
	Months []string `json:"months,omitempty" yaml:"months,omitempty"`
	// Years accepts range strings such as "2024:2026".
	Years []string `json:"years,omitempty" yaml:"years,omitempty"`
	// Location is an IANA timezone string, e.g. "Europe/Paris".
	Location string `json:"location,omitempty" yaml:"location,omitempty"`
}

// TimeRangeDef holds a start/end time as "HH:MM" strings (24-hour clock).
type TimeRangeDef struct {
	StartTime string `json:"start_time" yaml:"start_time"`
	EndTime   string `json:"end_time"   yaml:"end_time"`
}

// ─── Receivers ───────────────────────────────────────────────────────────────

// ReceiverDef is the API/JSON model for an Alertmanager receiver.
// The receiver name must be unique within the configuration.
type ReceiverDef struct {
	Name             string               `json:"name"                        yaml:"name"`
	WebhookConfigs   []WebhookConfigDef   `json:"webhook_configs,omitempty"   yaml:"webhook_configs,omitempty"`
	SlackConfigs     []SlackConfigDef     `json:"slack_configs,omitempty"     yaml:"slack_configs,omitempty"`
	EmailConfigs     []EmailConfigDef     `json:"email_configs,omitempty"     yaml:"email_configs,omitempty"`
	PagerdutyConfigs []PagerdutyConfigDef `json:"pagerduty_configs,omitempty" yaml:"pagerduty_configs,omitempty"`
	OpsgenieConfigs  []OpsgenieConfigDef  `json:"opsgenie_configs,omitempty"  yaml:"opsgenie_configs,omitempty"`
}

// WebhookConfigDef configures a generic HTTP webhook receiver.
type WebhookConfigDef struct {
	URL          string `json:"url"                    yaml:"url"`
	SendResolved bool   `json:"send_resolved,omitempty" yaml:"send_resolved,omitempty"`
	MaxAlerts    int    `json:"max_alerts,omitempty"    yaml:"max_alerts,omitempty"`
}

// SlackConfigDef configures a Slack receiver.
type SlackConfigDef struct {
	APIURL       string `json:"api_url,omitempty"       yaml:"api_url,omitempty"`
	Channel      string `json:"channel"                 yaml:"channel"`
	Username     string `json:"username,omitempty"      yaml:"username,omitempty"`
	Text         string `json:"text,omitempty"          yaml:"text,omitempty"`
	Title        string `json:"title,omitempty"         yaml:"title,omitempty"`
	SendResolved bool   `json:"send_resolved,omitempty" yaml:"send_resolved,omitempty"`
}

// EmailConfigDef configures an email receiver.
type EmailConfigDef struct {
	To           string `json:"to"                      yaml:"to"`
	From         string `json:"from,omitempty"          yaml:"from,omitempty"`
	Smarthost    string `json:"smarthost,omitempty"     yaml:"smarthost,omitempty"`
	AuthUsername string `json:"auth_username,omitempty" yaml:"auth_username,omitempty"`
	AuthPassword string `json:"auth_password,omitempty" yaml:"auth_password,omitempty"`
	SendResolved bool   `json:"send_resolved,omitempty" yaml:"send_resolved,omitempty"`
}

// PagerdutyConfigDef configures a PagerDuty receiver.
type PagerdutyConfigDef struct {
	RoutingKey   string `json:"routing_key,omitempty"   yaml:"routing_key,omitempty"`
	ServiceKey   string `json:"service_key,omitempty"   yaml:"service_key,omitempty"`
	Description  string `json:"description,omitempty"   yaml:"description,omitempty"`
	SendResolved bool   `json:"send_resolved,omitempty" yaml:"send_resolved,omitempty"`
}

// OpsgenieConfigDef configures an OpsGenie receiver.
type OpsgenieConfigDef struct {
	APIKey       string `json:"api_key,omitempty"       yaml:"api_key,omitempty"`
	Message      string `json:"message,omitempty"       yaml:"message,omitempty"`
	Priority     string `json:"priority,omitempty"      yaml:"priority,omitempty"`
	SendResolved bool   `json:"send_resolved,omitempty" yaml:"send_resolved,omitempty"`
}

// ─── Routes ──────────────────────────────────────────────────────────────────

// RouteSpec is the API/JSON model for an Alertmanager routing tree node.
// It mirrors the YAML Route structure and supports recursive child routes.
//
// Matchers uses the native Alertmanager string syntax, e.g.:
//
//	["severity=\"critical\"", "env=~\"prod.*\""]
type RouteSpec struct {
	Receiver            string      `json:"receiver,omitempty"              yaml:"receiver,omitempty"`
	GroupBy             []string    `json:"group_by,omitempty"              yaml:"group_by,omitempty"`
	Matchers            []string    `json:"matchers,omitempty"              yaml:"matchers,omitempty"`
	Continue            bool        `json:"continue,omitempty"              yaml:"continue,omitempty"`
	GroupWait           string      `json:"group_wait,omitempty"            yaml:"group_wait,omitempty"`
	GroupInterval       string      `json:"group_interval,omitempty"        yaml:"group_interval,omitempty"`
	RepeatInterval      string      `json:"repeat_interval,omitempty"       yaml:"repeat_interval,omitempty"`
	MuteTimeIntervals   []string    `json:"mute_time_intervals,omitempty"   yaml:"mute_time_intervals,omitempty"`
	ActiveTimeIntervals []string    `json:"active_time_intervals,omitempty" yaml:"active_time_intervals,omitempty"`
	Routes              []RouteSpec `json:"routes,omitempty"                yaml:"routes,omitempty"`
}
