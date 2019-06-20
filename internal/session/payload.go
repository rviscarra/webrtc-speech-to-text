package session

type newSessionRequest struct {
	Offer string `json:"offer"`
}

type newSessionResponse struct {
	Answer string `json:"answer"`
}
