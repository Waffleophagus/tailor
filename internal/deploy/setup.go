package deploy

// SetupInfo tells the UI that Tailscale is not configured for this deployment.
type SetupInfo struct {
	Required bool        `json:"required"`
	Hints    []SetupHint `json:"hints,omitempty"`
}

// SetupHint is a single setup instruction shown in the UI.
type SetupHint struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}
