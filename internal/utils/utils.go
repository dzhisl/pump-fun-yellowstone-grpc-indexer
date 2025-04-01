package utils

import (
	"github.com/gagliardetto/solana-go"
)

func GetPumpTokenAccounts(tokenMint solana.PublicKey) (bondingCurve, associatedBondingCurve *solana.PublicKey, err error) {
	seeds := [][]byte{
		[]byte("bonding-curve"),
		tokenMint.Bytes(),
	}
	bondingCurveAddress, _, err := solana.FindProgramAddress(seeds, solana.MustPublicKeyFromBase58("6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"))
	if err != nil {
		return nil, nil, err
	}

	associatedBCurve, _, err := solana.FindAssociatedTokenAddress(bondingCurveAddress, tokenMint)
	if err != nil {
		return nil, nil, err
	}
	return &bondingCurveAddress, &associatedBCurve, nil
}
