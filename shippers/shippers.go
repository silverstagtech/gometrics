package shippers

// Shipper is used to send measurements onto where ever they need to go
type Shipper interface {
	Ship([]byte)
}
