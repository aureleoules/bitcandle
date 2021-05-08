package electrum

import (
	"github.com/aureleoules/go-electrum/electrum"
)

var Client = electrum.NewServer()

func Connect(addr string) error {
	return Client.ConnectTCP(addr)
}
