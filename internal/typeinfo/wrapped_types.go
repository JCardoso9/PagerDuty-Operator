package typeinfo

import "github.com/PagerDuty/go-pagerduty"

// AlertGroupingParameters defines how alerts on the service will be automatically grouped into incidents
// +kubebuilder:validation:Enum=time;intelligent;content_based;null
type AlertGroupingParameters struct {
	Type   string                     `json:"type,omitempty"`
	Config *K8sAlertGroupParamsConfig `json:"config,omitempty"`
}

// AlertGroupParamsConfig is the config object on alert_grouping_parameters
type K8sAlertGroupParamsConfig pagerduty.AlertGroupParamsConfig
