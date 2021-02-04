# Notes
Things that still need to be figured out or implemented to make the IPFS WASM runtime a reality:
- How to support multiple outputs for a function
    - We currently support multiple arguments for a function, but I still haven't figured out the
    way of generalizing multiple outputs. However, this is possible, is just a matter of following the
    [multi example here](https://pkg.go.dev/github.com/bytecodealliance/wasmtime-go).
- A function manifest with all the information required to fetch ANY data, with ANY codec, and to run 
functions that can return ANY number of arguments of ANY types. _lots of ANYs to figure out_
- A function compiler to express the subsequence of functions in the network that wants to be run to
perform a complex task.