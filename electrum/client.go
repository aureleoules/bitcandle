package electrum

import (
	"github.com/checksum0/go-electrum/electrum"
)

var Client = electrum.NewServer()

func Connect(addr string) error {
	return Client.ConnectTCP(addr)
}
