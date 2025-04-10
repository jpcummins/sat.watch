package http

type errorJSON struct {
	Reason string `json:"reason"`
}

var errorInternal = errorJSON{Reason: "An internal error occurred."}
