package adapters

func NewKeyringSystem() *KeyringSystem {
	return &KeyringSystem{
		serviceName: shieldServiceName,
		keyName: shieldKeyName,
	}
}