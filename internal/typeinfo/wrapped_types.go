package typeinfo

import "github.com/PagerDuty/go-pagerduty"

type K8sAutoPauseNotificationsParameters struct {
	// +kubebuilder:default=false
	Enabled bool `json:"enabled"`
	// +kubebuilder:validation:Enum=120;180;240;300;600;900
	Timeout uint `json:"timeout,omitempty"`
}

type K8sSupportHours pagerduty.SupportHours

type K8sIncidentUrgencyType pagerduty.IncidentUrgencyType

type K8sIncidentUrgencyRule struct {
	Type                string                  `json:"type,omitempty"`
	Urgency             string                  `json:"urgency,omitempty"`
	DuringSupportHours  *K8sIncidentUrgencyType `json:"during_support_hours,omitempty"`
	OutsideSupportHours *K8sIncidentUrgencyType `json:"outside_support_hours,omitempty"`
}

// InlineModel represents when a scheduled action will occur.
type K8sInlineModel pagerduty.InlineModel

// ScheduledAction contains scheduled actions for the service.
type ScheduledAction struct {
	Type      string         `json:"type,omitempty"`
	At        K8sInlineModel `json:"at,omitempty"`
	ToUrgency string         `json:"to_urgency"`
}

// AlertGroupingParameters defines how alerts on the service will be automatically grouped into incidents
// +kubebuilder:validation:Enum=time;intelligent;content_based;null
type AlertGroupingParameters struct {
	Type   string                     `json:"type,omitempty"`
	Config *K8sAlertGroupParamsConfig `json:"config,omitempty"`
}

// AlertGroupParamsConfig is the config object on alert_grouping_parameters
type K8sAlertGroupParamsConfig pagerduty.AlertGroupParamsConfig
