package adapters

import (
	"fmt"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	dbusDest = "org.freedesktop.secrets"
	dbusPath = "/org/freedesktop/secrets"
)

type LinuxKeyring struct {
	conn       *dbus.Conn
	collection string
	itemLabel  string
}

func NewKeyringSystem() *LinuxKeyring {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		os.Exit(1)
	}
	return &LinuxKeyring{
		conn:       conn,
		collection: "shield-cli",
		itemLabel:  "master-key",
	}
}

func (l *LinuxKeyring) GetKey() ([]byte, error) {
	collectionPath, err := l.getDefaultCollection()
	if err != nil {
		return nil, fmt.Errorf("failed to find collection: %w", err)
	}

	serviceObj := l.conn.Object("org.freedesktop.secrets", "/org/freedesktop/secrets")
	sessionPath, err := l.openSession(serviceObj)
	if err != nil {
		return nil, fmt.Errorf("session error: %w", err)
	}
	defer l.closeSession(sessionPath)

	collectionObj := l.conn.Object("org.freedesktop.secrets", collectionPath)

	var items []dbus.ObjectPath
	err = collectionObj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.Secret.Collection", "Items",
	).Store(&items)

	if err != nil {
		return nil, fmt.Errorf("Failed to get items: %v", err)
	}

	for _, itemPath := range items {
		itemObj := l.conn.Object("org.freedesktop.secrets", itemPath)

		attrs := make(map[string]string)
		itemObj.Call("org.freedesktop.DBus.Properties.Get", 0,
			"org.freedesktop.Secret.Item", "Attributes",
		).Store(&attrs)

		if attrs["keyname"] != "master-key" {
			continue
		}

		secret := struct {
			Session     dbus.ObjectPath
			Parameters  []byte
			Value       []byte
			ContentType string
		}{Session: sessionPath}

		err = itemObj.Call("org.freedesktop.Secret.Item.GetSecret", 0, sessionPath).Store(&secret)
		if err != nil {
			continue
		}

		return secret.Value, nil
	}

	return nil, fmt.Errorf("Master key not found")
}

func (l *LinuxKeyring) SaveKey(key []byte) error {
	collectionPath, err := l.getDefaultCollection()
	if err != nil {
		return fmt.Errorf("failed to find collection: %w", err)
	}

	serviceObj := l.conn.Object("org.freedesktop.secrets", "/org/freedesktop/secrets")

	sessionPath, err := l.openSession(serviceObj)
	if err != nil {
		return fmt.Errorf("session error: %w", err)
	}
	defer l.closeSession(sessionPath)

	collectionObj := l.conn.Object("org.freedesktop.secrets", collectionPath)

	itemProps := map[string]dbus.Variant{
		"org.freedesktop.Secret.Item.Label": dbus.MakeVariant("Shield Master Key"),
		"org.freedesktop.Secret.Item.Attributes": dbus.MakeVariant(map[string]string{
			"xdg:schema": "shield:master-key",
			"keyname":    "master-key",
		}),
	}

	secret := struct {
		Session     dbus.ObjectPath
		Parameters  []byte
		Value       []byte
		ContentType string
	}{
		Session:     sessionPath,
		Parameters:  []byte{},
		Value:       key,
		ContentType: "application/octet-stream",
	}

	var createdItem dbus.ObjectPath
	var promptPath dbus.ObjectPath
	err = collectionObj.Call("org.freedesktop.Secret.Collection.CreateItem", 0,
		itemProps,
		secret,
		true,
	).Store(&createdItem, &promptPath)

	if err != nil {
		return fmt.Errorf("Failed to save item: %v", err)
	}

	if promptPath != "/" {
		_, err := l.handlePrompt(promptPath)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Key saved at: %s\n", createdItem)
	return nil
}

func (l *LinuxKeyring) getDefaultCollection() (dbus.ObjectPath, error) {
	serviceObj := l.conn.Object("org.freedesktop.secrets", "/org/freedesktop/secrets")

	var collections []dbus.ObjectPath
	err := serviceObj.Call("org.freedesktop.DBus.Properties.Get", 0,
		"org.freedesktop.Secret.Service", "Collections",
	).Store(&collections)

	if err != nil {
		return "", fmt.Errorf("Failed to get collections: %v", err)
	}

	preferredCollections := []string{"kdewallet", "login", "default"}
	var preferredPath dbus.ObjectPath

	for _, prefName := range preferredCollections {
		for _, collPath := range collections {
			if strings.HasSuffix(string(collPath), "/"+prefName) {
				preferredPath = collPath
				break
			}
		}
		if preferredPath != "" {
			break
		}
	}

	if preferredPath == "" {
		for _, collPath := range collections {
			if !strings.HasSuffix(string(collPath), "/Shield") &&
				!strings.HasPrefix(string(collPath), "/org/freedesktop/secrets/collection/Shield__") {
				preferredPath = collPath
				break
			}
		}
	}

	if preferredPath == "" && len(collections) > 0 {
		preferredPath = collections[0]
	}

	var unlockedPaths []dbus.ObjectPath
	var promptPath dbus.ObjectPath
	unlockCall := serviceObj.Call("org.freedesktop.Secret.Service.Unlock", 0,
		[]dbus.ObjectPath{preferredPath},
	)
	err = unlockCall.Store(&unlockedPaths, &promptPath)
	if err != nil {
		return "", fmt.Errorf("Failed to unlock collection: %v", err)
	}

	if promptPath != "/" {
		_, err = l.handlePrompt(promptPath)
		if err != nil {
			return "", err
		}
	}

	if len(unlockedPaths) > 0 {
		return unlockedPaths[0], nil
	}

	return preferredPath, nil
}

func (l *LinuxKeyring) openSession(serviceObj dbus.BusObject) (dbus.ObjectPath, error) {
	var sessionPath dbus.ObjectPath
	var output dbus.Variant

	err := serviceObj.Call("org.freedesktop.Secret.Service.OpenSession", 0,
		"plain",
		dbus.MakeVariant(""),
	).Store(&output, &sessionPath)

	if err != nil {
		return "", fmt.Errorf("Failed to open session: %v", err)
	}

	return sessionPath, nil
}

func (l *LinuxKeyring) closeSession(sessionPath dbus.ObjectPath) {
	sessionObj := l.conn.Object("org.freedesktop.secrets", sessionPath)
	sessionObj.Call("org.freedesktop.Secret.Session.Close", 0)
}

func (l *LinuxKeyring) handlePrompt(promptPath dbus.ObjectPath) (dbus.Variant, error) {
	if promptPath == "/" {
		return dbus.Variant{}, nil
	}

	ch := make(chan *dbus.Signal, 10)
	l.conn.Signal(ch)

	defer l.conn.RemoveMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.Secret.Prompt"),
		dbus.WithMatchMember("Completed"),
		dbus.WithMatchObjectPath(promptPath),
	)

	err := l.conn.AddMatchSignal(
		dbus.WithMatchInterface("org.freedesktop.Secret.Prompt"),
		dbus.WithMatchMember("Completed"),
		dbus.WithMatchObjectPath(promptPath),
	)

	if err != nil {
		return dbus.Variant{}, fmt.Errorf("erro ao registrar sinal: %v", err)
	}

	obj := l.conn.Object("org.freedesktop.secrets", promptPath)
	err = obj.Call("org.freedesktop.Secret.Prompt.Prompt", 0, "").Err

	if err != nil {
		return dbus.Variant{}, fmt.Errorf("Failed to emit prompt: %v", err)
	}

	fmt.Println("Waiting human interaction...")
	for sig := range ch {
		if sig.Path == promptPath && sig.Name == "org.freedesktop.Secret.Prompt.Completed" {
			dismissed := sig.Body[0].(bool)
			result := sig.Body[1].(dbus.Variant)

			if dismissed {
				return dbus.Variant{}, fmt.Errorf("User cancelled!")
			}
			return result, nil
		}
	}
	return dbus.Variant{}, fmt.Errorf("Prompt closed!")
}

func (l *LinuxKeyring) Lock() error {
	collectionPath, err := l.getDefaultCollection()
	if err != nil {
		return fmt.Errorf("failed to find collection to lock: %w", err)
	}

	serviceObj := l.conn.Object(dbusDest, dbusPath)

	var lockedPaths []dbus.ObjectPath
	var promptPath dbus.ObjectPath

	return serviceObj.Call("org.freedesktop.Secret.Service.Lock", 0,
		[]dbus.ObjectPath{collectionPath},
	).Store(&lockedPaths, &promptPath)
}
