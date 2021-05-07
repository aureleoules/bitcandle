package consensus

// P2SHPushDataLimit represents the maximum size of data that can be pushed on the stack at a time in a Bitcoin script
const P2SHPushDataLimit = 520

// P2PKHDustLimit represents the minimum amount of sats a public key hash output can receive
const P2PKHDustLimit = 546

// P2SHTotalDataLimit represents the total amount of data that can be stored in a script signature
const P2SHTotalDataLimit = 1461
