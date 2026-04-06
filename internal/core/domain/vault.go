package domain

type AuthMethod string

const (
	AuthMethodPassword AuthMethod = "password"
	AuthMethodKey AuthMethod = "key"
	NoneAuthMethod AuthMethod = "none"
)

type SSHEntry struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port int `json:"port"`
	User string `json:"user"`
	AuthType AuthMethod `json:"auth_type"`

	Password string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
}

type Vault struct {
	Entries map[string]SSHEntry `json:"entries"`
}

func NewVault() Vault {
	return Vault{
		Entries: make(map[string]SSHEntry),
	}
}
