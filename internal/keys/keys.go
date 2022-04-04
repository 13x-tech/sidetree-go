package keys

import (
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

func SignatureAlgorithmFromKey(k jwk.Key) (jwa.SignatureAlgorithm, error) {
	if k == nil {
		return "", fmt.Errorf("key is nil")
	}

	if k.KeyType() == jwa.RSA {
		return jwa.PS256, nil
	}

	// if k.KeyType() == jwa.EC {
	// 	switch k.Algorithm() {

	// 	}
	// }

	return "", nil
}
