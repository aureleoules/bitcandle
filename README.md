# Bitcandle
Bitcandle allows you to store arbitrary data on Bitcoin **very** efficiently up to 83 kB by using P2SH signature scripts.

## Disclaimer
Storing data on Bitcoin is a controversial subject as it bloats the blockchain with _somewhat unnecessary_ data.  
I built this tool to show that there exists efficient methods of storing data, lesser known than standard P2PKH / P2PK / OP_RETURN methods.

## Usage
```bash
Usage:
  bitcandle [flags]
  bitcandle [command]

Available Commands:
  help        Help about any command
  inject      Inject a file on the Bitcoin network
  retrieve    Retrieve a file on the Bitcoin network

Flags:
  -h, --help   help for bitcandle

Use "bitcandle [command] --help" for more information about a command.
```
### Example
#### Inject data
```bash
$ ./bitcandle inject \
    --fee 1 \
    --file ./image.jpg \
    --network testnet \
    --change-address bcrt1q3yqqaenczxs9uytz55c0xhdrunqd2efq2wlnaq

✔ Loaded 3072 bytes to inject.
⚠ File is too large (> 1461 bytes) for a single input.
✔ Loaded private key.
✔ Connected to electrum server (localhost:50001).
ℹ Estimated injection cost: 0.00004307 BTC.
ℹ You must send 0.00001436 BTC to 2N1tyf6uJvpgup147SUEGjKi3dYpQXBqtNY.
ℹ You must send 0.00001436 BTC to 2N2NcYkDrc9A4F9gFtajg8dxgNMfe8EmLPF.
ℹ You must send 0.00001436 BTC to 2N9U5GHc6XC3FfrTy9M3LDLfB4yFWTE1Mgx.
ℹ Copy paste this in Electrum -> Tools -> Pay to many.

2N1tyf6uJvpgup147SUEGjKi3dYpQXBqtNY,0.00001436
2N2NcYkDrc9A4F9gFtajg8dxgNMfe8EmLPF,0.00001436
2N9U5GHc6XC3FfrTy9M3LDLfB4yFWTE1Mgx,0.00001436

✔ Payment received. (1/3)
✔ Payment received. (2/3)
✔ Payment received. (3/3)
✔ All payments received.
✔ Data injected.
ℹ TxID: 006a6f058bd756eb5693eb5319bcdc3c78649b9efd3beeb2a8f85e6841b9ce21
```

#### Retrieve data
```bash
$ ./bitcandle retrieve \
    --tx 006a6f058bd756eb5693eb5319bcdc3c78649b9efd3beeb2a8f85e6841b9ce21 \
    --network testnet \
    -o /tmp/data

✔ Connected to electrum server (localhost:50001).
✔ Retrieved file.
✔ Saved file to "/tmp/data".
```

## Docker
```bash
$ mkdir data
$ cp ~/Downloads/image.jpg ./data/
$ docker run -it --rm -v $PWD/data:/data aureleoules/bitcandle inject -f ./image.jpg [args]
...
$ docker run -it --rm -v $PWD/data:/data aureleoules/bitcandle retrieve [args]
...
```

## Cost
Before injecting data we need to create P2SH UTXOs. This can be done in a single transaction, sent by the user, by sending coins to many outputs.   
To spend the P2SH UTXOs (injecting data), it costs roughly **0.000011940 BTC per kB**. (**1 sat/B** fee rate)  
It is _obviously_ free to retrieve data at any given time.

## How it works
This tool uses script signatures in P2SH inputs to store data.  
Instead of storing data in the transaction outputs like standard approaches such as P2PKH, P2PK or OP_RETURN, this tool stores data in the transaction inputs.  

### P2SH redeem script
We need to create a new P2SH UTXO. To do that we need to create a redeem script hash address.  
The redeem script needs to look like this:  
- OP_HASH160
- OP_PUSHDATA [CHUNK 3 HASH]
- OP_EQUALVERIFY 
- OP_HASH160
- OP_PUSHDATA [CHUNK 2 HASH]
- OP_EQUALVERIFY
- OP_HASH160
- OP_PUSHDATA [CHUNK 1 HASH]
- OP_EQUALVERIFY
- OP_PUSHDATA [PUBKEY]
- OP_CHECKSIG

This redeem script is hashed and wrapped in a P2SH output script to create a P2SH address such as: 3N9fEcf9yUSspvUc78cQQVJDQi5NkgrHtLQ.  

The user must send enough funds to this address so that this UTXO can be spent.  

Hashes of chunks are pushed on the stack in order to ensure data integrity.  
Once we spend this UTXO, at attacker could scramble chunks of data and the transaction would this be valid if these op codes were not added.  

We must also add the PUBKEY and the CHECKSIG op code so that transactions outputs are signed. This prevents attackers from redirecting the output change to another change address. This may not be necessary for small change amounts (minimum on mainnet is 546 sats) but it is recommended as it makes sure the transaction id does not change while the transaction is in the mempool.  

This redeem script ensures data integrity and prevents output sniping.  

### P2SH script signature
In order to spend the UTXO and essentially store the file, we must create a script signature that can unlock the redeem script built previously.  
It looks something like this:  
* OP_PUSHDATA [SIG]
* OP_PUSHDATA [CHUNK 1]
* OP_PUSHDATA [CHUNK 2]
* OP_PUSHDATA [CHUNK 3]
* OP_PUSHDATA [redeem script hex]

Chunks of data pushed using the PUSHDATA op code can only fit a maximum of 520 bytes which means the redeem script can only be 520 bytes.  
So we are only able to store 1461 bytes per UTXO. For larger files, multiple UTXOs must be created using different redeem scripts.

### Notes
Redeem scripts are built deterministically such that for the same file and same public key, the P2SH addresses will remain the same. This may help easily retrieving any stuck funds if needed.  
Not a single satoshi is burned in the data injection process. This is a clear advantage compared to other known injection methods like P2PKH. All the fees go back to miners and the change is sent to the address specified.

## Use cases
This software can be useful to
* store censorship restistant documents
* store pictures of loved ones we wish to remember forever
* store documents publicly and permanently accessible

## License
[MIT](https://github.com/aureleoules/bitcandle/blob/master/LICENSE) - [Aurèle Oulès](https://www.aureleoules.com)
