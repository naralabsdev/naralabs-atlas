package scval

import (
	"math/big"

	"github.com/stellar/go-stellar-sdk/xdr"
)

func u128Decimal(parts xdr.UInt128Parts) string {
	n := new(big.Int).SetUint64(uint64(parts.Hi))
	n.Lsh(n, 64)
	n.Or(n, new(big.Int).SetUint64(uint64(parts.Lo)))
	return n.String()
}

func i128Decimal(parts xdr.Int128Parts) string {
	n := big.NewInt(int64(parts.Hi))
	n.Lsh(n, 64)
	n.Or(n, new(big.Int).SetUint64(uint64(parts.Lo)))
	return n.String()
}

func u256Decimal(parts xdr.UInt256Parts) string {
	n := new(big.Int).SetUint64(uint64(parts.HiHi))
	for _, limb := range []uint64{uint64(parts.HiLo), uint64(parts.LoHi), uint64(parts.LoLo)} {
		n.Lsh(n, 64)
		n.Or(n, new(big.Int).SetUint64(limb))
	}
	return n.String()
}

func i256Decimal(parts xdr.Int256Parts) string {
	n := big.NewInt(int64(parts.HiHi))
	for _, limb := range []uint64{uint64(parts.HiLo), uint64(parts.LoHi), uint64(parts.LoLo)} {
		n.Lsh(n, 64)
		n.Or(n, new(big.Int).SetUint64(limb))
	}
	return n.String()
}
