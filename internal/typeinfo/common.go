package typeinfo

import (
	"fmt"

	"github.com/PagerDuty/go-pagerduty"
)

const (
	MinimumEscalationRuleTargets = 1
	MaximumEscalationRuleTargets = 10
)

func ConvertTargets(ids []UserID) ([]pagerduty.APIObject, error) {
	if len(ids) < MinimumEscalationRuleTargets || len(ids) > MaximumEscalationRuleTargets {
		return nil, fmt.Errorf("invalid number of targets: %d", len(ids))
	}
	targets := make([]pagerduty.APIObject, len(ids))
	for i, id := range ids {
		targets[i] = id.ToAPIObject()
	}
	return targets, nil
}

type Converter interface {
	Convert() interface{}
}

//////////////////////////////////////// Creating objects from IDs ////////////////////////////////////////

type UserID string

type UserIDList []UserID

type EscalationPolicyID string

type TeamID string

type Teams []TeamID

type ReferenceCreator interface {
	ToReference() pagerduty.APIReference
}

func (id EscalationPolicyID) ToReference() pagerduty.APIReference {
	return pagerduty.APIReference{
		ID:   string(id),
		Type: "escalation_policy_reference",
	}
}

func (id TeamID) ToReference() pagerduty.APIReference {
	return pagerduty.APIReference{
		ID:   string(id),
		Type: "team_reference",
	}
}

func (id UserID) ToReference() pagerduty.APIReference {
	return pagerduty.APIReference{
		ID:   string(id),
		Type: "user_reference",
	}
}

func (ids UserIDList) ToReference() []pagerduty.APIReference {
	refs := make([]pagerduty.APIReference, len(ids))
	for i, ids := range ids {
		refs[i] = ids.ToReference()
	}
	return refs
}

func (teams Teams) ToReference() []pagerduty.APIReference {
	refs := make([]pagerduty.APIReference, len(teams))
	for i, teams := range teams {
		refs[i] = teams.ToReference()
	}
	return refs
}

type ObjectCreator interface {
	ToAPIObject() pagerduty.APIReference
}

func (id EscalationPolicyID) ToAPIObject() pagerduty.APIObject {
	return pagerduty.APIObject{
		ID:   string(id),
		Type: "escalation_policy_reference",
	}
}

func (id TeamID) ToAPIObject() pagerduty.APIObject {
	return pagerduty.APIObject{
		ID:   string(id),
		Type: "team_reference",
	}
}

func (id UserID) ToAPIObject() pagerduty.APIObject {
	return pagerduty.APIObject{
		ID:   string(id),
		Type: "user_reference",
	}
}

func (ids UserIDList) ToAPIObject() []pagerduty.APIObject {
	refs := make([]pagerduty.APIObject, len(ids))
	for i, ids := range ids {
		refs[i] = ids.ToAPIObject()
	}
	return refs
}

func (teams Teams) ToAPIObject() []pagerduty.APIObject {
	refs := make([]pagerduty.APIObject, len(teams))
	for i, teams := range teams {
		refs[i] = teams.ToAPIObject()
	}
	return refs
}

type SpecificObjectCreator interface {
	ToSpecificObject() interface{}
}

func (id EscalationPolicyID) ToSpecificObject() pagerduty.EscalationPolicy {
	return pagerduty.EscalationPolicy{
		APIObject: pagerduty.APIObject{
			ID:   string(id),
			Type: "escalation_policy_reference",
		},
	}
}

func (id TeamID) ToSpecificObject() pagerduty.Team {
	return pagerduty.Team{
		APIObject: pagerduty.APIObject{
			ID:   string(id),
			Type: "team_reference",
		},
	}
}

func (id UserID) ToSpecificObject() pagerduty.User {
	return pagerduty.User{
		APIObject: pagerduty.APIObject{
			ID:   string(id),
			Type: "user_reference",
		},
	}
}

///// Comparison functions

func (id UserID) compareAPIObject(apiObject pagerduty.APIObject) bool {
	return string(id) == apiObject.ID && "user_reference" == apiObject.Type
}

func (ids UserIDList) compareAPIObject(apiObject []pagerduty.APIObject) bool {
	if len(ids) != len(apiObject) {
		return false
	}
	for i, id := range ids {
		if !id.compareAPIObject(apiObject[i]) {
			return false
		}
	}
	return true
}
