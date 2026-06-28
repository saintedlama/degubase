package identity

// CreateWorkspaceRequest is the body for POST /workspaces.
type CreateWorkspaceRequest struct {
	Name    string `json:"name"    example:"My Workspace"`
	Context string `json:"context" example:"Project context"`
	Code    string `json:"code"    example:"MYWS"`
}

// UpdateWorkspaceRequest is the body for PUT /workspaces/{workspaceCode}.
type UpdateWorkspaceRequest struct {
	Name       string `json:"name"        example:"My Workspace"`
	Context    string `json:"context"     example:"Project context"`
	MCPEnabled *bool  `json:"mcp_enabled"`
}

// LoginRequest is the body for POST /auth/login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateUserRequest is the body for POST /admin/users.
type CreateUserRequest struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

// CreateTokenRequest is the body for POST /workspaces/{workspaceCode}/tokens.
type CreateTokenRequest struct {
	Name string `json:"name"`
}
