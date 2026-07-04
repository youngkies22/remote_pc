package storage

// Store mengelompokkan seluruh repository berbasis file agar mudah di-inject
// ke server melalui dependency injection.
type Store struct {
	Devices *DeviceRepo
	Users   *UserRepo
}

// Open membuka seluruh repository di bawah dataDir. Direktori dibuat bila perlu.
func Open(dataDir string) (*Store, error) {
	devices, err := NewDeviceRepo(dataDir)
	if err != nil {
		return nil, err
	}
	users, err := NewUserRepo(dataDir)
	if err != nil {
		return nil, err
	}
	return &Store{Devices: devices, Users: users}, nil
}
