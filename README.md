# SideTree Go

A focused Go library implementing the **read / resolution** half of the
[Sidetree protocol](https://identity.foundation/sidetree/spec/): given a Bitcoin
anchor string of the form `<operationCount>.<coreIndexFileURI>`, it walks the
Sidetree file graph stored in content-addressable storage (CAS, e.g. IPFS) and
reconstructs the typed DID operations contained in an anchored batch.

It processes the spec-ordered file chain:

```
Core Index File → Core Proof File → Provisional Index File → Provisional Proof File → Chunk File
```

and enforces the protocol's structural rules: no duplicate DID suffix within a
batch, proof/index entry-count and index-position alignment, single-chunk
(v1), and positional mapping of chunk deltas back onto the concatenated
`create → recover → update` operation arrays.

The DID/operation data models (`did.SuffixData`, `did.Delta`,
`operations.*`) come from
[`ion-sdk-go`](https://github.com/13x-tech/ion-sdk-go); cryptographic
verification of the signed recover/update/deactivate proofs is performed by that
package's operation-replay engine. This package attaches the signed data to the
operations it resolves.

## Usage

```go
cas := myCAS{} // implement the CAS interface (Start/Get/Put/Close/Type), backed by IPFS

st, err := sidetree.New(
    sidetree.WithCAS(cas),
    sidetree.WithPrefix("ion"),
)
if err != nil {
    log.Fatal(err)
}

// anchors are typically discovered from Bitcoin OP_RETURN outputs by a node.
results := st.ProcessOperations(anchors, nil /* optional DID filter */)
```

The library ships **no concrete CAS implementation** — consumers (e.g. a full
node) supply one backed by IPFS. `CAS.Get`/`Put` are expected to transparently
gunzip/gzip content.

## Status

Builds and tests on Go 1.23 (`go test ./...`, ~99.7% statement coverage,
including an end-to-end processor test). Known gaps tracked as issues:
spec `MAX_*` file-size enforcement, and `ietf-json-patch` support (in
`ion-sdk-go`).

This package is intended to be one small piece of a larger SideTree toolkit.

## License
[Apache-2](https://www.apache.org/licenses/LICENSE-2.0)

© Copyright 2022, 13x LLC.
