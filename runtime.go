package ipfslite

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/fxamacker/cbor"
	"github.com/ipfs/go-cid"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// TODO: All of this data structures should be determined in IPLD.

// Type of the data. Expressed with a name and a codec to encode/decode.
type Type struct {
	Name  string
	Codec cid.Cid
}

// // Data defined by its type and where the content is stored.
// type Data struct {
// 	Type    cid.Cid
// 	Content cid.Cid
// }

// FxABI function interface.
type FxABI struct {
	// TODO: The FXs definition should be more complex to include their signature so they can return
	// more than one value.
	Fxs      []string // Name of the functions
	Bytecode cid.Cid  // We could add a type here if we want to support several runtimes.
	Args     []Type
}

func (f *FxABI) Encode() (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	err := cbor.NewEncoder(buf, cbor.CanonicalEncOptions()).Encode(f)
	return buf, err
}

func (f *FxABI) Decode(r io.Reader) error {
	err := cbor.NewDecoder(r).Decode(&f)
	return err
}

// Deploy a function to the network.
func (p *Peer) Deploy(ctx context.Context, fxs []string, bytecode []byte, args []Type) (*cid.Cid, error) {
	// TODO: Add an IPLD DAG instead of chunking files directly.
	bytecodeCid, err := p.AddFile(ctx, bytes.NewReader(bytecode), &AddParams{})
	fmt.Println("Bytecode deployed at: ", bytecodeCid)
	if err != nil {
		return nil, err
	}
	abi := FxABI{
		Fxs:      fxs,
		Bytecode: bytecodeCid.Cid(),
		Args:     args,
	}
	// TODO: Use CBOR encoding better. As I abandoned IPLD because it was taking me too much
	// will look into this while IPLD is supported.
	// b, err := abi.Encode()
	b, err := json.Marshal(abi)
	if err != nil {
		return nil, err
	}
	root, err := p.AddFile(ctx, bytes.NewReader(b), &AddParams{})
	rootCid := root.Cid()
	fmt.Println("ABI deployed at: ", rootCid)

	return &rootCid, err
}

// Call a function deployed in the network
func (p *Peer) Call(ctx context.Context, fnCid cid.Cid, fxName string, argsCid []cid.Cid) (*cid.Cid, error) {
	abi := FxABI{}
	// Get the manifest.
	rsc, err := p.GetFile(ctx, fnCid)
	if err != nil {
		return nil, err
	}
	defer rsc.Close()
	d, err := ioutil.ReadAll(rsc)
	err = json.Unmarshal(d, &abi)
	// TODO: This depends on the encoding used for the ABI
	//abi.Decode(rsc)
	if err != nil {
		return nil, err
	}

	cids := append([]cid.Cid{abi.Bytecode}, argsCid...)
	data := make([][]byte, 0)

	// Get the required data from the network.
	for _, c := range cids {
		rsc, err := p.GetFile(ctx, c)
		if err != nil {
			return nil, err
		}
		defer rsc.Close()
		d, err := ioutil.ReadAll(rsc)
		if err != nil {
			return nil, err
		}
		data = append(data, d)
	}

	module, err := wasmtime.NewModule(p.runtime.Engine, data[0])

	instance, err := wasmtime.NewInstance(p.runtime, module, []*wasmtime.Extern{})
	if err != nil {
		return nil, err
	}

	call32 := func(f *wasmtime.Func, args ...interface{}) (int32, error) {
		ret, err := f.Call(args...)
		if err != nil {
			return 0, err
		}
		return ret.(int32), nil
	}

	// Instantiate memory
	memory := instance.GetExport("memory").Memory()
	// Alloc function
	alloc := instance.GetExport("alloc").Func()
	// Function to call
	fx := instance.GetExport(fxName).Func()

	linearInput := []byte{}
	argOffsets := make([]interface{}, 0)
	// Concatenate inputs linearly
	for _, k := range data[1:] {
		linearInput = append(linearInput, k...)
		// Offsets to get parameters inside WASM.
		argOffsets = append(argOffsets, len(k))
	}

	// Allocating extra 100 just in case.
	a, err := call32(alloc, len(linearInput)+100)
	if err != nil {
		fmt.Println("Error calling Wasm function")
		return nil, err
	}
	buf := memory.UnsafeData()

	// Allocate arguments in memory
	for i, k := range linearInput {
		buf[int(a)+i] = k
	}

	// Prepare arguments putting allocated pointer first
	args := append([]interface{}{a}, argOffsets...)

	b, err := call32(fx, args...)
	if err != nil {
		fmt.Println("Error calling Wasm function")
		return nil, err
	}

	// Add cid to the network.
	output, err := p.AddFile(ctx, bytes.NewReader(buf[a:a+b]), &AddParams{})
	outputCid := output.Cid()

	if err != nil {
		return nil, err
	}
	return &outputCid, nil

}

func checkArgs(args []string, l int) error {
	if len(args) != l {
		fmt.Println("Wrong number of arguments")
		return fmt.Errorf("Wrong number of arguments")
	}
	return nil
}

// ExecCmd from script
func (p *Peer) ExecCmd(ctx context.Context, text string, done chan bool) error {
	text = strings.ReplaceAll(text, "\n", "")
	// TODO: Do not trim spaces for now.
	// text = strings.ReplaceAll(text, " ", "")
	words := strings.Split(text, "_")

	// Defer notifying the that processing is finished.
	if done != nil {
		defer func() {
			done <- true
		}()
	}

	if words[0] == "exit" {
		os.Exit(0)
	}
	if words[0] == "help" {
		helpcmd()
		return nil
	}
	if len(words) < 2 {
		fmt.Println("Wrong number of arguments")
		return fmt.Errorf("Wrong number of arguments")
	}

	// If we use add we can add random content to the network.
	if words[0] == "addFile" {
		bytecode, err := ioutil.ReadFile(words[1])
		if err != nil {
			fmt.Println("Couldn't read file: ", err)
			return err
		}

		cid, err := p.AddFile(ctx, bytes.NewReader(bytecode), nil)
		if err != nil {
			fmt.Println("Couldn't add file to IPFS: ", err)
			return err
		}
		fmt.Println("Added file with CID: ", cid)

	} else if words[0] == "add" {
		bytecode := []byte(words[1])
		cid, err := p.AddFile(ctx, bytes.NewReader(bytecode), nil)
		if err != nil {
			fmt.Println("Couldn't add string to IPFS: ", err)
			return err
		}
		fmt.Println("Added string with CID: ", cid)

	} else if words[0] == "get" {
		c, err := cid.Decode(string(words[1]))
		if err != nil {
			fmt.Println("Couldn't parse CID: ", err)
			return err
		}
		rsc, err := p.GetFile(ctx, c)
		d, err := ioutil.ReadAll(rsc)
		if err != nil {
			fmt.Println("Couldn't get file: ", err)
			return err
		}
		fmt.Println("Get: ", string(d))

	} else if words[0] == "abi" {

		c, err := cid.Decode(string(words[1]))
		if err != nil {
			fmt.Println("Couldn't parse CID: ", err)
			return err
		}
		rsc, err := p.GetFile(ctx, c)
		d, err := ioutil.ReadAll(rsc)
		abi := FxABI{}
		err = json.Unmarshal(d, &abi)
		// TODO: Change according to the encoding used for ABIs, ideally IPLD
		// err = abi.Decode(rsc)
		if err != nil {
			fmt.Println("Couldn't decode ABI: ", err)
			return err
		}
		fmt.Println("ABI: ", abi)
	} else if words[0] == "deploy" {
		if e := checkArgs(words, 4); e != nil {
			return e
		}
		bytecode, err := ioutil.ReadFile(words[1])
		if err != nil {
			fmt.Println("Couldn't read file: ", err)
			return err
		}

		fxIn := strings.Split(words[2], "&")
		argsIn := strings.Split(words[3], "&")

		args := []Type{}
		for _, k := range argsIn {
			args = append(args, Type{Name: k})
		}

		cid, err := p.Deploy(ctx, fxIn, bytecode, args)
		if err != nil {
			fmt.Println("Couldn't deploy to IPFS: ", err)
			return err
		}
		fmt.Println("Deployed function at: ", cid)

	} else if words[0] == "connect" {
		p.connectCmd(ctx, words[1])

	} else if words[0] == "script" {
		// fmt.Println("This is still a work in progress...")
		err := p.RunScript(ctx, words[1])
		if err != nil {
			return err
		}

	} else if words[0] == "get" {
		c, err := cid.Decode(string(words[1]))
		if err != nil {
			fmt.Println("Couldn't parse CID: ", err)
			return err
		}
		rsc, err := p.GetFile(ctx, c)
		d, err := ioutil.ReadAll(rsc)
		if err != nil {
			fmt.Println("Couldn't get file: ", err)
			return err
		}
		fmt.Println("Get: ", string(d))

	} else if words[0] == "call" {
		if e := checkArgs(words, 4); e != nil {
			return e
		}
		cids := []cid.Cid{}

		fnCid, err := cid.Decode(string(words[1]))
		if err != nil {
			fmt.Println("Couldn't parse CID: ", err)
			return err
		}

		argsIn := strings.Split(words[3], "&")
		// Parse CIDs
		for _, cs := range argsIn {
			c, err := cid.Decode(string(cs))
			if err != nil {
				fmt.Println("Couldn't parse CID: ", err)
				return err
			}
			cids = append(cids, c)
		}

		out, err := p.Call(ctx, fnCid, words[2], cids)
		if err != nil {
			fmt.Println("Couldn't run function: ", err)
			return err
		}
		fmt.Println("Output CID: ", out.String())

	} else {
		fmt.Println("[!] Wrong command")
		helpcmd()
	}

	return nil
}

func helpcmd() {
	fmt.Println(`[!] Commands available:
	* script_<file_dir>
	* addFile_<file_dir>
	* add_<string>
	* get_<cid>
	* abi_<cid>
	* deploy_<bytecode>_<fn1>&<fn2>_<typeArg1>&<typeArg2>
	* connect_<peer_multiaddr>
	* call_<fxCid>_<fxname>_<argCid1>&<argCid2>
	* exit`)
}

// conectPeer connects to a peer in the network.
func (p *Peer) connectCmd(ctx context.Context, id string) error {
	maddr, err := ma.NewMultiaddr(id)
	if err != nil {
		fmt.Println("Invalid peer ID")
		return err
	}
	fmt.Println("Multiaddr", maddr)
	addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		fmt.Println("Invalid peer info", err)
		return err
	}
	err = p.ConnectPeer(*addrInfo)
	if err != nil {
		fmt.Println("Couldn't connect to peer", err)
		return err
	}
	fmt.Println("Connected successfully to peer")
	return nil
}

// RunScript IP Language
func (p *Peer) RunScript(ctx context.Context, script string) error {

	file, err := os.Open(script)
	if err != nil {
		fmt.Println("Couldn't run script", err)
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		in := scanner.Text()
		if in[0] != '#' {
			err = p.ExecCmd(ctx, in, nil)
			if err != nil {
				return err
			}
		}
		fmt.Println(in)
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
