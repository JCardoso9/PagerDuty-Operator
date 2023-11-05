package typeinfo

// APIObject represents generic api json response that is shared by most
// domain objects (like escalation)
type APIObject struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type,omitempty"`
	Summary string `json:"summary,omitempty"`
	Self    string `json:"self,omitempty"`
	HTMLURL string `json:"html_url,omitempty"`
}

// APIListObject are the fields used to control pagination when listing objects.
type APIListObject struct {
	Limit  uint `json:"limit,omitempty"`
	Offset uint `json:"offset,omitempty"`
	More   bool `json:"more,omitempty"`
	Total  uint `json:"total,omitempty"`
}

// APIReference are the fields required to reference another API object.
type APIReference struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
}

// APIDetails are the fields required to represent a details non-hydrated
// object.
type APIDetails struct {
	Type    string `json:"type,omitempty"`
	Details string `json:"details,omitempty"`
}
