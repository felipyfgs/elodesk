package dto

// ParticipantResp is the shape returned by
// GET /api/v1/accounts/:aid/conversations/:id/participants for group
// conversations. Wraps a fully hydrated ContactResp under the embedded role.
type ParticipantResp struct {
	ID      int64       `json:"id"`
	Role    string      `json:"role"`
	Contact ContactResp `json:"contact"`
}

// ParticipantListResp is the envelope used by the GET endpoint above.
type ParticipantListResp struct {
	Data []ParticipantResp `json:"data"`
}
