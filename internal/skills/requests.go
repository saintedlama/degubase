package skills

// CreateSkillRequest is the body for creating a skill.
type CreateSkillRequest struct {
	Name        string   `json:"name"        example:"My Skill"`
	Description string   `json:"description" example:"Access workspace data"`
	TableIDs    []int64  `json:"table_ids"`
	Operations  []string `json:"operations"  example:"list,create,update,delete"`
	TokenName   string   `json:"token_name"  example:"my-skill-token"`
}

// UpdateSkillRequest is the body for updating a skill.
type UpdateSkillRequest struct {
	Name        string   `json:"name"        example:"My Skill"`
	Description string   `json:"description" example:"Access workspace data"`
	TableIDs    []int64  `json:"table_ids"`
	Operations  []string `json:"operations"  example:"list,create,update,delete"`
}

// PreviewSkillRequest is the body for generating a skill preview without persisting.
type PreviewSkillRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TableIDs    []int64  `json:"table_ids"`
	Operations  []string `json:"operations"`
}
