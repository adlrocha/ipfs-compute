# IPFS Compute: Run your CID!
> [IPFS-lite](github.com/hsanjuan/ipfs-lite) node with an embedded WASM runtime to run your CIDs!
_This is a PoC to explore the idea of computing over IPFS, do not expect production-ready code (at leas for now)._

<p align="center">
<img src="logo.png" alt="ipfs-lite" title="ipfs-lite" />
</p>

IPFS compute will allow you to run WASM functions stored in the network and identified with a CID, over data in the
network ALSO identified with a CID, putting the result in YET ANOTHER CID. In its current PoC implementation,
there is no support por IPLD graphs, complex codecs or structures. However, all the conceptual foundations are there
to use common interfaces to self-describe the code and the data.

All of these ideas have some interesting consequences:
* Self-describing code and data. Using a weird codec for your data? No problem, just point to the codec's code in the network.
* Need to do computations over a large dataset? Don't wait to download de data to perform the computation. Get the computation near the data
just request the result.
* Collaborative computation. Partial results are stored in the network in the form of CIDs, so other peer can pick it up and resume the computation.

In the long-term, this simple idea opens the door to other impactful things such as:
* A IPLD-based programming language that targets a Universal Runtime (in this proposal, WASM).
* IPFS as an operating system. IPFS is a decentralized File System, and this Universal Runtime offers the
computational resources required to run the operating system.
    * _"I personally love the idea of seamless UX between all of my devices. If my data and my code live in the
    network, there is no need to worry about 'where I left that file' anymore"_


### IPFS Compute over IPFS-Lite
IPFS-Lite is an embeddable, lightweight IPFS peer which runs the minimal setup
to provide an `ipld.DAGService`. It can:

* Add, Get, Remove IPLD Nodes to/from the IPFS Network (remove is a local blockstore operation).
* Add single files (chunk, build the DAG and Add) from a `io.Reader`.
* Get single files given a their CID.
* And now with its embeddded runtime, it can run WASM loads stored in the network over data ALSO stored in the network.

It needs:

* A [libp2p Host](https://godoc.org/github.com/libp2p/go-libp2p#New)
* A [libp2p DHT](https://godoc.org/github.com/libp2p/go-libp2p-kad-dht#New)
* A [datastore](https://godoc.org/github.com/ipfs/go-datastore) like [BadgerDB](https://godoc.org/github.com/ipfs/go-ds-badger)

Some helper functions are provided to
[initialize these quickly](https://godoc.org/github.com/hsanjuan/ipfs-lite#SetupLibp2p).

It provides:

* An [`ipld.DAGService`](https://godoc.org/github.com/ipfs/go-ipld-format#DAGService)
* An [`AddFile` method](https://godoc.org/github.com/hsanjuan/ipfs-lite#Peer.AddFile) to add content from a reader
* A [`GetFile` method](https://godoc.org/github.com/hsanjuan/ipfs-lite#Peer.GetFile) to get a file from IPFS.

The goal of IPFS-Lite is to run the **bare minimal** functionality for any
IPLD-based application with an embedded runtime 
to interact with the IPFS Network by getting and
putting blocks to it, running WASM loads from the network over that
in the network, and placing the result in the network.

### Getting started
To get started clone this repo and run the cli:

```sh
$ git clone https://github.com/adlrocha/ipfs-compute
$ cd cli
$ go run cli
```

The CLI will allow you interact with the node seamlessly. Try running the following commands in the CLI:

```sh
# Deploy a WASM module
>> Enter command: deploy_/home/adlrocha/Desktop/main/work/ProtocolLabs/repos/ipruntime/ipfs-computation/functions/simple.wasm_fx_string
Bytecode deployed at:  bafybeidob3ooeaysa3x3ahwwgzwkzcqiiegxb33jul53xpsl3vyeqmkmey
ABI deployed at:  bafybeihxh2j47fwociwwl6whvsebfk554p4fua5nssejvftnexgrpsnswi
Deployed function at:  bafybeihxh2j47fwociwwl6whvsebfk554p4fua5nssejvftnexgrpsnswi
# Check the module's ABI
>>  Enter command: abi_bafybeihxh2j47fwociwwl6whvsebfk554p4fua5nssejvftnexgrpsnswi
ABI:  {[fx] bafybeidob3ooeaysa3x3ahwwgzwkzcqiiegxb33jul53xpsl3vyeqmkmey [{string b}]}
# Add a string to the network
>> Enter command: add_HelloWorld!
Added string with CID:  bafybeigft6kodyhajn5m2fx6raevfcj6umdob2jiuntufbvttwpygz767q
# Run fx function from WASM module
>> Enter command: call_bafybeihxh2j47fwociwwl6whvsebfk554p4fua5nssejvftnexgrpsnswi_fx_bafybeigft6kodyhajn5m2fx6raevfcj6umdob2jiuntufbvttwpygz767q
Output CID:  bafybeifxbaroablgiucsnahwd7jjyrabwx6nbm6w27g33rf7qgrm53bwsi
# Get the result from the network.
get_bafybeifxbaroablgiucsnahwd7jjyrabwx6nbm6w27g33rf7qgrm53bwsi
Get:  HelloWorld! <-- Tagged from Wasm
```
Cool, right? In the `functions` directory you will find another WASM load that has two functions,
`map` and `reduce` to count words with a map-reduce approach (allowing you to check the partial
computation capabilities of IPFS-compute). See if you can make it work by yourself so you start playing
with it, and if you can't drop me an issue.

The CLI provides a help command to see all the avialable commands:
```sh
>>  Enter command: help
[!] Commands available:
        * script_<file_dir>
        * addFile_<file_dir>
        * add_<string>
        * get_<cid>
        * abi_<cid>
        * deploy_<bytecode>_<fn1>&<fn2>_<typeArg1>&<typeArg2>
        * connect_<peer_multiaddr>
        * call_<fxCid>_<fxname>_<argCid1>&<argCid2>
        * exit
```

### The Interpreter
In an attempt to also explore the idea of having a programming language that understand IPFS, I leveraged the
CLI code to build an "intereter" (disclaimer: this does not even remotely resemble anything such as an interpreter 
or a programming language, but it was a fun thing to try to show how a scripting language that understand CID could work).
I had many more ideas that I couldn't explore but will hopefully do in the near future.
```sh
>> Enter command: script_../interpreter/script.ipl
# test Script
Bytecode deployed at:  bafybeihizlepdz3rt25l3dhdnh4vgk35vwjhs66a4fuu3eqisidb2ynq5i
ABI deployed at:  bafybeifc27yhx3fvcpwdgaclhmls5rnbyxjwcuxtqyvfhb4om6qmfds23e
Deployed function at:  bafybeifc27yhx3fvcpwdgaclhmls5rnbyxjwcuxtqyvfhb4om6qmfds23e
deploy_/home/adlrocha/Desktop/main/work/ProtocolLabs/repos/ipruntime/ipfs-computation/functions/wordcount.wasm_map&reduce_string
ABI:  {[map reduce] bafybeihizlepdz3rt25l3dhdnh4vgk35vwjhs66a4fuu3eqisidb2ynq5i [{string b}]}
abi_bafybeifc27yhx3fvcpwdgaclhmls5rnbyxjwcuxtqyvfhb4om6qmfds23e
Added string with CID:  bafybeifwriigj6u6462fbrnoy2wbmuqckrsmppl5ixfwtea6hgk7knhi5y
add_We come from the land of the ice and snow,
Added string with CID:  bafybeidodg5jkps5vtoxpgxi4ixdo5alosqrnhhlgvjyjj2nyuj2dmckou
add_From the midnight sun where the hot springs flow.
Output CID:  bafybeibzebwnm5dg5dxcoizzesvsscjgnk4ek2mnkqg23hccrfeikqv4ce
call_bafybeifc27yhx3fvcpwdgaclhmls5rnbyxjwcuxtqyvfhb4om6qmfds23e_map_bafybeifwriigj6u6462fbrnoy2wbmuqckrsmppl5ixfwtea6hgk7knhi5y
Output CID:  bafybeifb2ysundozrn4jctamfejm4zjylsap2mjw66sqlenhvhqvgas7yi
call_bafybeifc27yhx3fvcpwdgaclhmls5rnbyxjwcuxtqyvfhb4om6qmfds23e_map_bafybeidodg5jkps5vtoxpgxi4ixdo5alosqrnhhlgvjyjj2nyuj2dmckou
Output CID:  bafybeiabbonc433eyn2fteixyemm6k2f7xda63kzoedkqae7tvqbxpm7lu
call_bafybeifc27yhx3fvcpwdgaclhmls5rnbyxjwcuxtqyvfhb4om6qmfds23e_reduce_bafybeibzebwnm5dg5dxcoizzesvsscjgnk4ek2mnkqg23hccrfeikqv4ce&bafybeifb2ysundozrn4jctamfejm4zjylsap2mjw66sqlenhvhqvgas7yi
Get:  {"x":{"of":1,"the":4,"sun":1,"and":1,"from":2,"midnight":1,"we":1,"ice":1,"come":1,"where":1,"land":1,"springs":1,"snow":1,"flow":1,"hot":1}}
get_bafybeiabbonc433eyn2fteixyemm6k2f7xda63kzoedkqae7tvqbxpm7lu
```
You can also run scripts going to:
```sh
cd interpreter
go run interpreter.go ./script.ipl
```


## License

Apache 2.0
