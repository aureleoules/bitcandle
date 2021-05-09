package consensus

// P2SHP2WSHPushDataLimit represents the maximum size of data that can be pushed on the stack at a time
const P2SHP2WSHPushDataLimit = 80

// P2SHP2WSHStackItems represents the maximum amount of items that can be pushed in the witness script
const P2SHP2WSHStackItems = 100

// P2PKHDustLimit represents the minimum amount of sats a public key hash output can receive
const P2PKHDustLimit = 546

// ECDSAMaxSignatureSize represents the maximum size of an ECDSA signature
const ECDSAMaxSignatureSize = 73

// BTCSats represents a Bitcoin in sats
const BTCSats = 100_000_000
