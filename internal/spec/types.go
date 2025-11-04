package spec

// ProductSpec represents the complete product specification
type ProductSpec struct {
	Product       string        `json:"product"`
	Goals         []string      `json:"goals"`
	Features      []Feature     `json:"features"`
	NonFunctional NonFunctional `json:"non_functional"`
	Acceptance    []string      `json:"acceptance"`
	Milestones    []Milestone   `json:"milestones"`
}

// Feature represents a single feature in the product spec
type Feature struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Desc     string   `json:"desc"`
	Priority string   `json:"priority"` // P0, P1, P2
	API      []API    `json:"api,omitempty"`
	Success  []string `json:"success"`
	Trace    []string `json:"trace"`
}

// API represents an API endpoint definition
type API struct {
	Method   string `json:"method"`
	Path     string `json:"path"`
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
}

// NonFunctional represents non-functional requirements
type NonFunctional struct {
	Performance  []string `json:"performance,omitempty"`
	Security     []string `json:"security,omitempty"`
	Scalability  []string `json:"scalability,omitempty"`
	Availability []string `json:"availability,omitempty"`
}

// Milestone represents a development milestone
type Milestone struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	FeatureIDs  []string `json:"feature_ids"`
	TargetDate  string   `json:"target_date,omitempty"`
	Description string   `json:"description,omitempty"`
}

// SpecLock represents the canonical, hashed specification snapshot
type SpecLock struct {
	Version  string                    `json:"version"`
	Features map[string]LockedFeature  `json:"features"`
}

// LockedFeature represents a feature with its hash and generated artifacts
type LockedFeature struct {
	Hash        string   `json:"hash"`          // blake3(canonical feature JSON)
	OpenAPIPath string   `json:"openapi_path"`
	TestPaths   []string `json:"test_paths"`
}
