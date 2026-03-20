package auth

// SSOProvider is the interface each SSO backend must implement.
type SSOProvider interface {
	// AuthURL returns the redirect URL for the SSO login page.
	AuthURL(state string) string
	// ExchangeUser exchanges the callback code for a normalized UserInfo.
	ExchangeUser(code string) (*UserInfo, error)
}

type UserInfo struct {
	ID    string
	Email string
	Name  string
}

// TODO: implement WechatWorkProvider, LDAPProvider, SAMLProvider