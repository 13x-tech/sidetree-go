package main

import (
	"fmt"

	"github.com/13x-tech/sidetree-go"
)

func main() {
	fmt.Printf("Generating Identity")

	key1, key2, err := sidetree.GenerateKeys()
	if err != nil {
		panic(err)
	}
	services := []sidetree.DIDService{
		{
			ID:   "linkeddomains",
			Type: "LinkedDomains",
			ServiceEndpoint: map[string]interface{}{
				"origins": []string{"https://www.linkeddomains.com"},
			},
		},
	}

	did, err := sidetree.NewDID(
		sidetree.WithGenerateKeys(),
		sidetree.WithPubKeys(key1, key2),
		sidetree.WithServices(services...),
	)

	if err != nil {
		panic(err)
	}

	longform, err := did.LongFormURI()
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n\nDID: %s\n", longform)

}
