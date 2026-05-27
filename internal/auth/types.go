package auth

type Profile struct {
	ID          string            `json:"id"`
	ProviderKey string            `json:"providerKey"`
	AuthMode    string            `json:"authMode"`
	DisplayName string            `json:"displayName"`
	Token       string            `json:"token,omitempty"`
	Cookie      string            `json:"cookie,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
	Status      string            `json:"status"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

type Validation struct {
	ID          string `json:"id"`
	ProfileID   string `json:"profileId"`
	ProviderKey string `json:"providerKey"`
	OK          bool   `json:"ok"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	CreatedAt   string `json:"createdAt"`
}
