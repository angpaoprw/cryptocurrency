package api

type Alchemy struct {
	APIKey  string
	BaseURL string
}

func NewAlchemy(apiKey string) *Alchemy {
	return &Alchemy{
		APIKey:  apiKey,
		BaseURL: "https://eth-mainnet.alchemyapi.io/v2/",
	}
}

func (a *Alchemy) GetURL() string {
	return a.BaseURL + a.APIKey
}
