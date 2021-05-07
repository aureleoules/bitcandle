package consensus

// P2SHPushDataLimit represents the maximum size of data that can be pushed on the stack at a time in a Bitcoin script
const P2SHPushDataLimit = 520

// P2PKHDustLimit represents the minimum amount of sats a public key hash output can receive
const P2PKHDustLimit = 546

// P2SHInputDataLimit represents the total amount of data that can be stored in a script signature
const P2SHInputDataLimit = 1461

// ECDSAMaxSignatureSize represents the maximum size of an ECDSA signature
const ECDSAMaxSignatureSize = 73

// BTCSats represents a Bitcoin in sats
const BTCSats = 100_000_000
