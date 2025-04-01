package parser

import (
	"fmt"
	"time"

	"github.com/dzhisl/geyser-converter/shared"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"go.uber.org/zap"
)

const (
	pumpProgram      = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"
	selfCPIAuthority = "Ce6TQqeHC9p8KetsN6JsjHK7UTZk7nasjjnr7XxXp9F1"
)

var (
	NewTokensChan = make(chan *PumpFunCreation, 1)
	NewSwapsChan  = make(chan *AnchorSelfCPILogSwapData, 5)
)

type AnchorSelfCPILogSwapData struct {
	Discriminator        [16]byte
	Mint                 [32]byte
	SolAmount            uint64
	TokenAmount          uint64
	IsBuy                bool
	User                 [32]byte
	Timestamp            int64
	VirtualSolReserves   uint64
	VirtualTokenReserves uint64
	Signature            string `borsh_skip:"true" json:"signature"`
}

type PumpFunCreation struct {
	ID           uint             `gorm:"primaryKey" borsh_skip:"true"`
	Name         string           `json:"name"`
	Symbol       string           `json:"symbol"`
	Uri          string           `json:"uri"`
	MintAddress  solana.PublicKey `json:"mint"`
	BondingCurve solana.PublicKey `json:"bondingCurve"`
	Creator      solana.PublicKey `json:"user"`
	Signature    string           `borsh_skip:"true" json:"signature"`
	CreatedAt    time.Time        `borsh_skip:"true"`
}

func (m *AnchorSelfCPILogSwapData) Decode(decodedData []byte) error {
	return bin.NewBorshDecoder(decodedData).Decode(m)
}

func (m *PumpFunCreation) Decode(in []byte) error {
	return bin.NewBorshDecoder(in[16:]).Decode(m)
}

func ParseSelfCpiLog(txDetails *shared.TransactionDetails) error {
	for _, inst := range txDetails.Instructions {
		if len(inst.Accounts) == 12 && inst.ProgramID.PublicKey == pumpProgram {
			if d, err := parseTradeEvent(inst); err == nil {
				d.Signature = txDetails.Signature
				NewSwapsChan <- d
			} else {
				zap.L().Error("error parsing trade event", zap.Error(err))
			}
		} else if len(inst.Accounts) == 14 && inst.ProgramID.PublicKey == pumpProgram {
			zap.L().Debug("token create tx", zap.String("signature", txDetails.Signature))
			if d, err := parseTokenCreation(inst); err == nil {
				d.Signature = txDetails.Signature
				NewTokensChan <- d
			} else {
				zap.L().Error("error parsing token creation", zap.Error(err))
			}
		}

		for _, innerInst := range inst.InnerInstructions {
			if innerInst.ProgramID.PublicKey != pumpProgram {
				continue
			}

			if len(innerInst.Accounts) == 12 && innerInst.ProgramID.PublicKey == pumpProgram {
				if d, err := parseTradeEvent(inst); err == nil {
					d.Signature = txDetails.Signature
					NewSwapsChan <- d
				} else {
					zap.L().Error("error parsing trade event", zap.Error(err))
				}
			} else if len(innerInst.Accounts) == 14 && innerInst.ProgramID.PublicKey == pumpProgram {
				zap.L().Debug("token create tx", zap.String("signature", txDetails.Signature))
				if d, err := parseTokenCreation(inst); err == nil {
					d.Signature = txDetails.Signature
					NewTokensChan <- d
				} else {
					zap.L().Error("error parsing token creation", zap.Error(err))
				}
			}
		}
	}
	return nil
}

func parseTokenCreation(inst shared.InstructionDetails) (*PumpFunCreation, error) {
	for _, innerInst := range inst.InnerInstructions {
		if len(innerInst.Accounts) == 1 && innerInst.Accounts[0].PublicKey == selfCPIAuthority {
			var createEvent PumpFunCreation
			if err := createEvent.Decode(innerInst.Data); err != nil {
				return nil, fmt.Errorf("error decoding create token event: %w", err)
			}
			return &createEvent, nil
		}
	}
	return nil, nil
}

func parseTradeEvent(inst shared.InstructionDetails) (*AnchorSelfCPILogSwapData, error) {
	for _, innerInst := range inst.InnerInstructions {
		if len(innerInst.Accounts) == 1 && innerInst.Accounts[0].PublicKey == selfCPIAuthority {
			var tradeEvent AnchorSelfCPILogSwapData
			if err := tradeEvent.Decode(innerInst.Data); err != nil {
				return nil, fmt.Errorf("error decoding trade token event: %w", err)
			}
			return &tradeEvent, nil
		}
	}
	return nil, nil
}
