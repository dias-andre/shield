package adapters

func NewKeyringSystem() (*KeyringSystem, error) {
	return &KeyringSystem{
		serviceName: shieldServiceName,
		keyName:     shieldKeyName,
	}, nil
}
