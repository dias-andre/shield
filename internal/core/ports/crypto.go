package ports

type EncryptorPort interface {
	Encrypt(vault []byte, key []byte) ([]byte, error)
	Decrypt(data []byte, key []byte) ([]byte, error)
}