package ports

type KeySystemPort interface {
	GetKey() ([]byte, error)
	SaveKey([]byte) error
}