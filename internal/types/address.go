package types

import "math/big"

type AddressState struct {
	Height   int64
	Received *big.Int
	Sent     *big.Int
}
