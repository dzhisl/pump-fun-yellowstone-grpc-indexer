package database

import (
	"context"
	"log"
	"os"
	"pf-indexer/internal/parser"
	"pf-indexer/internal/utils"
	"time"

	"github.com/mr-tron/base58"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const (
	//REPLACE

	maxIdleConns       = 5
	maxOpenConns       = 10
	connMaxLifetime    = time.Hour
	slowQueryThreshold = 2000 * time.Millisecond
)

var dbURL string

var DB *gorm.DB

// Swap represents a swap transaction
type Swap struct {
	ID          uint   `gorm:"primaryKey"`
	Account     string `gorm:"not null;index"`
	Mint        string `gorm:"not null;index"`
	SolAmount   uint64 `gorm:"not null"`
	TokenAmount uint64 `gorm:"not null"`
	IsBuy       bool   `gorm:"not null;index"`
	CreatedAt   time.Time
	Signature   string `gorm:"not null"`
}

// Pool represents a liquidity pool
type Pool struct {
	ID                     uint   `gorm:"primaryKey"`
	Mint                   string `gorm:"unique;not null;index"`
	BondingCurve           string `gorm:"not null"`
	AssociatedBondingCurve string `gorm:"not null"`
	VirtualSolReserves     uint64 `gorm:"not null"`
	VirtualTokenReserves   uint64 `gorm:"not null"`
	CreatedAt              time.Time
	Signature              string `gorm:"not null"`
	LastUpdated            time.Time
}

type PumpFunCreation struct {
	ID           uint      `gorm:"primaryKey" borsh_skip:"true"`
	Name         string    `json:"name"`
	Symbol       string    `json:"symbol"`
	Uri          string    `json:"uri"`
	MintAddress  string    `json:"mint"`
	BondingCurve string    `json:"bondingCurve"`
	Creator      string    `json:"user"`
	Signature    string    `borsh_skip:"true" json:"signature"`
	CreatedAt    time.Time `borsh_skip:"true"`
}

// InitDB initializes the database connection with connection pooling
func InitDB() error {
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	// Construct the DB URL from environment variables
	if dbHost == "" || dbUser == "" || dbPassword == "" || dbName == "" || dbPort == "" || dbSSLMode == "" {
		log.Fatal("Missing one or more required environment variables for database connection")
	}

	dbURL = "host=" + dbHost +
		" user=" + dbUser +
		" password=" + dbPassword +
		" dbname=" + dbName +
		" port=" + dbPort +
		" sslmode=" + dbSSLMode
	// Initialize DB connection
	newLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             slowQueryThreshold,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
	var err error
	DB, err = gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger:      newLogger,
		PrepareStmt: true, // Enable prepared statement cache
	})
	if err != nil {
		return err
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	// Set connection pool parameters
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	// Auto-migrate models
	if err := DB.AutoMigrate(&Swap{}, &Pool{}, &PumpFunCreation{}); err != nil {
		log.Fatal(err)
	}

	DB.Logger.Info(context.Background(), "Database connection established successfully")
	return nil
}

// ToSwap converts parser data to database model
func toSwap(p parser.AnchorSelfCPILogSwapData) Swap {
	return Swap{
		Account:     base58.Encode(p.User[:]),
		Mint:        base58.Encode(p.Mint[:]),
		SolAmount:   p.SolAmount,
		TokenAmount: p.TokenAmount,
		IsBuy:       p.IsBuy,
		CreatedAt:   time.Now(),
		Signature:   p.Signature,
	}
}

func toNewTokenCreation(p parser.PumpFunCreation) PumpFunCreation {
	return PumpFunCreation{
		Name:         p.Name,
		Symbol:       p.Symbol,
		Uri:          p.Uri,
		MintAddress:  p.MintAddress.String(),
		BondingCurve: p.BondingCurve.String(),
		Creator:      p.Creator.String(),
		Signature:    p.Signature,
		CreatedAt:    p.CreatedAt,
	}
}

func toPool(p parser.AnchorSelfCPILogSwapData) Pool {
	b, ab, err := utils.GetPumpTokenAccounts(p.Mint)
	if err != nil {
		log.Fatal(err)
	}
	return Pool{
		Mint:                   base58.Encode(p.Mint[:]),
		BondingCurve:           b.String(),
		AssociatedBondingCurve: ab.String(),
		VirtualSolReserves:     p.VirtualSolReserves,
		VirtualTokenReserves:   p.VirtualTokenReserves,
		CreatedAt:              time.Now(),
		Signature:              p.Signature,
		LastUpdated:            time.Now(),
	}
}

// AddSwap inserts a single swap record with context
func AddSwap(ctx context.Context, swap parser.AnchorSelfCPILogSwapData) error {
	s := toSwap(swap)
	return DB.WithContext(ctx).Create(&s).Error
}

func AddNewToken(ctx context.Context, token parser.PumpFunCreation) error {
	s := toNewTokenCreation(token)
	return DB.WithContext(ctx).Create(&s).Error
}

// AddOrUpdatePool adds or updates a pool record
func AddOrUpdatePool(ctx context.Context, pool parser.AnchorSelfCPILogSwapData) error {
	p := toPool(pool)
	return DB.WithContext(ctx).Clauses(
		clause.OnConflict{
			Columns: []clause.Column{{Name: "mint"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"bonding_curve",
				"associated_bonding_curve",
				"virtual_sol_reserves",
				"virtual_token_reserves",
				"signature",
				"last_updated",
			}),
		},
	).Create(&p).Error
}

// AddSwapsBatch inserts multiple swap records in a single batch operation
func AddSwapsBatch(ctx context.Context, swaps []parser.AnchorSelfCPILogSwapData) error {
	if len(swaps) == 0 {
		return nil
	}

	batch := make([]Swap, len(swaps))
	for i, swap := range swaps {
		batch[i] = toSwap(swap)
	}

	return DB.WithContext(ctx).Create(&batch).Error
}
