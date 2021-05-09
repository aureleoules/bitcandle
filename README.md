# Bitcandle
Store data on Bitcoin for 350 sats/KB up to 185 kB by using P2SH-P2WSH witness scripts.

[225ed8bc432d37cf434f80717286fd5671f676f12b573294db72a2a8f9b1e7ba](https://blockstream.info/tx/225ed8bc432d37cf434f80717286fd5671f676f12b573294db72a2a8f9b1e7ba)

## Disclaimer
Storing data on Bitcoin is a controversial subject as it bloats the blockchain with _somewhat unnecessary_ data.  
I built this tool to show that there exists an efficient method of storing data: P2SH-P2WSH, lesser known than standard P2PKH / P2PK / OP_RETURN methods.

## Cost
Before injecting data we need to create P2SH-P2WSH UTXOs. This can be done in a single transaction, sent by the user, by sending coins to many outputs.   
To spend the P2SH-P2WSH UTXOs (injecting data), it costs roughly **0.00000349 BTC per KB**. (**1 sat/B** fee rate).  
This is about **84 times** cheaper than the P2PKH injection method. 

It is _obviously_ free to retrieve data.

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
    --network mainnet \
    --change-address bc1q8sl9tnvnuc8z7q80u9wffdf9ugt4arrp6vlamg

✔ Loaded 4556 bytes to inject.
✔ Loaded existing private key.
✔ Connected to electrum server (blockstream.info:110).
ℹ Estimated injection cost: 0.00002176 BTC.
ℹ You must send 0.00002176 BTC to 33z4X8jkMd8WCzhrfgEigzgLyrap1ACWUE.
█████████████████████████████████████████
█████████████████████████████████████████
████ ▄▄▄▄▄ ██▀▄███▀  ▀▄ ▀ ▄ ▀█ ▄▄▄▄▄ ████
████ █   █ █▄▀█▄▀▀▄ ▀█▀▄▀█▀▀██ █   █ ████
████ █▄▄▄█ ██▄▀▀ ▄       █ ▀▀█ █▄▄▄█ ████
████▄▄▄▄▄▄▄█ █▄▀▄█▄▀▄▀ ▀ ▀ █ █▄▄▄▄▄▄▄████
████▄ ▄▀  ▄  ▀█▀ ▀███▄█▀ ▄▀█   ███ ▀▀████
████ ▀ ▄ █▄█ ▀ ▄ ▀ █▄ ▀▀▀ ▄█ █▀▀▀▄█▄ ████
████▄▄▄▀██▄▀ █  ██ ██ ▀▀▀ ██  ▄██▀▀▀ ████
█████▀ ▀▀▄▄▀ ▄▀▀█▄█▀▄▀▀█▀ ▀█▄▄█▀ ▄▄ ▄████
████▄▄ █▄▀▄▀▀█ ▀ ▀███▄█▀▀███▀   ██▀▄ ████
████▄▀▀▄ ▀▄███▀▄ ▀ ▀▄ ███▀█▀▄ ▄▀ ▀█▄▄████
████▄▄▄▀ ▀▄▀█▀█ ██ ██ ▀▀ ▄▀▄ █ ▄▄ ▀▀▀████
████▄▀▀▄ ▄▄▀  █▀█▄█▀▄▀█▀█▀ █▄▀██▄▀█  ████
████▄█▄███▄▄ █▄▀ ▀███▄ ▀  ▀▄ ▄▄▄  █▀ ████
████ ▄▄▄▄▄ █ ██▄ ▀ ▀▄▀▀▀█ ▄▄ █▄█  █ ▄████
████ █   █ █▄   ██ ██  ██ ██ ▄▄▄  █▄▀████
████ █▄▄▄█ █ █▀▀█▄██ ▀▄▀▄▀ ▀▄▄▄▀ ▄█▀▄████
████▄▄▄▄▄▄▄█▄█▄█▄██▄▄▄▄██▄█▄▄▄███▄█▄▄████
█████████████████████████████████████████
▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀
✔ Payment received. (1/1)
✔ All payments received.
✔ Data injected.
ℹ TxID: 225ed8bc432d37cf434f80717286fd5671f676f12b573294db72a2a8f9b1e7ba
```

#### Retrieve data
```bash
$ ./bitcandle retrieve \
    --tx 225ed8bc432d37cf434f80717286fd5671f676f12b573294db72a2a8f9b1e7ba \
    --network mainnet \
    -o /tmp/image.jpg

✔ Connected to electrum server (blockstream.info:110).
✔ Retrieved file.
✔ Saved file to "/tmp/image.jpg".
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

### Working example
```bash
$ docker run -it --rm -v $PWD/data:/data aureleoules/bitcandle \
    retrieve \
    --tx 225ed8bc432d37cf434f80717286fd5671f676f12b573294db72a2a8f9b1e7ba \
    -o /data/image.jpg
```

## How it works
This tool uses witness scripts in P2SH-P2WSH inputs to store data.  
Instead of storing data in the transaction outputs like standard approaches such as P2PKH, P2PK or OP_RETURN, this tool stores data in the transaction inputs (witness).  

### Witness script
We need to create a new P2SH-P2WSH UTXO. To do that we need to create a witness script.  
The script needs to look like this:  
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

This witness script is hashed and wrapped in a P2SH-P2WSH output script to create a P2SH-P2WSH address such as: 3N9fEcf9yUSspvUc78cQQVJDQi5NkgrHtLQ.  

The user must send enough funds to this address so that this UTXO can be spent.  

Hashes of chunks are pushed on the stack in order to ensure data integrity.  
Once we spend this UTXO, at attacker could scramble chunks of data and the transaction would this be valid if these op codes were not added.  

We must also add the PUBKEY and the CHECKSIG op code so that transactions outputs are signed. This prevents attackers from redirecting the output change to another change address. This may not be necessary for small change amounts (minimum on mainnet is 546 sats) but it is recommended as it makes sure the transaction id does not change while the transaction is in the mempool.  

This witness script ensures data integrity and prevents output sniping.  

### Witness data
In order to spend the UTXO and essentially store the file, we must include witness data that unlocks the witness script built previously.  
It looks something like this:  
* [SIG]
* [CHUNK 1]
* [CHUNK 2]
* [CHUNK 3]
* [witness script hex]

This is not a script, this is simply a stack of data. It can only fit 99 chunks of 80 bytes of data.  
For larger files, multiple UTXOs must be created using different witness scripts.

### Notes
Witness scripts are built deterministically such that for the same file and same public key, the P2SH-P2WSH addresses will remain the same. This may help easily retrieving any stuck funds if needed.  
Not a single satoshi is burned in the data injection process. This is a clear advantage compared to other known injection methods like P2PKH. All the fees go back to miners and the change is sent to the address specified.

## Use cases
This software can be useful to
* store censorship resistant documents
* store pictures of loved ones we wish to remember forever
* store documents publicly and permanently accessible

## License
[MIT](https://github.com/aureleoules/bitcandle/blob/master/LICENSE) - [Aurèle Oulès](https://www.aureleoules.com)
