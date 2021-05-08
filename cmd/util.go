package cmd

import (
	"io/ioutil"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
)

func loadChainParams(net Network) *chaincfg.Params {
	switch net {
	case Mainnet:
		return &chaincfg.MainNetParams
	case Testnet:
		return &chaincfg.TestNet3Params
	case RegressionTest:
		return &chaincfg.RegressionNetParams
	}

	return nil
}

func loadKey(path string) (*btcec.PrivateKey, error) {
	key, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), key)
	return pKey, nil
}

func getDefaultElectrumServer(network Network) string {
	switch network {
	case Mainnet:
		return "blockstream.info:110"
	case Testnet:
		return "blockstream.info:143"
	case RegressionTest:
		return "localhost:50001"
	}
	return ""
}
