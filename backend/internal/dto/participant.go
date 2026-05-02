package dto

type ParticipantResp struct {
	ID      int64       `json:"id"`
	Role    string      `json:"role"`
	Contact ContactResp `json:"contact"`
}

type ParticipantListResp struct {
	Data []ParticipantResp `json:"data"`
}
