package main

import (
	"context"
	"os"
	"time"

	"pf-indexer/database"
	logger "pf-indexer/internal/logger"
	"pf-indexer/internal/parser"

	geyserAdapter "github.com/dzhisl/geyser-converter/geyser"
	"github.com/dzhisl/geyser-converter/utils"
	"go.uber.org/zap"

	"github.com/mr-tron/base58"
	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
)

const (
	targetAccount = "6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"
)

var endpointURL = os.Getenv("GRPC_ENDPOINT")

func main() {
	logger.InitLogger()
	database.InitDB()
	time.Sleep(3 * time.Second)
	grpcAdapter := geyserAdapter.NewGeyserAdapter()

	conn, err := grpcAdapter.CreateGRPCConnection(endpointURL)
	if err != nil {
		zap.L().Fatal("Connection failed", zap.String("err", err.Error()))
	}
	defer conn.Close()

	client := pb.NewGeyserClient(conn)
	stream, err := client.Subscribe(context.Background())
	if err != nil {
		zap.L().Fatal("Failed to create stream", zap.Error(err))
	}

	if err := stream.Send(grpcAdapter.CreateSubscriptionRequest(targetAccount)); err != nil {
		zap.L().Fatal("Subscription failed", zap.Error(err))
	}

	zap.L().Info("🔭 Monitoring transactions for account")

	// Start batch processing in a separate goroutine
	go batchInsertWorker()

	for {
		update, err := stream.Recv()
		if err != nil {
			zap.L().Fatal("Stream error", zap.Error(err))
		}
		go processUpdate(update)
	}
}

func processUpdate(update *pb.SubscribeUpdate) {
	if update.GetTransaction() == nil {
		return
	}

	tx := update.GetTransaction()
	signature := tx.GetTransaction().GetSignature()
	if signature == nil {
		return
	}

	sigStr := base58.Encode(signature)
	txDetails, err := utils.ProcessTransactionToStruct(tx, sigStr)
	if err != nil {
		zap.L().Error("Error processing tx details", zap.Error(err))
		return
	}

	err = parser.ParseSelfCpiLog(txDetails)
	if err != nil {

		zap.L().Error("Error parsing logs", zap.Error(err))
		zap.L().Info(sigStr)
		return
	}

}
func batchInsertWorker() {
	var swapBatch []parser.AnchorSelfCPILogSwapData

	for {
		select {
		case token, ok := <-parser.NewTokensChan:
			if !ok {
				zap.L().Debug("Token channel closed")
				return
			}
			if token == nil {
				continue
			}

			zap.L().Debug("New Token Received", zap.String("signature", token.Signature), zap.Any("data", token))
			err := database.AddNewToken(context.Background(), *token)
			if err != nil {
				zap.L().Error("error adding new token to table", zap.Error(err))
				continue
			}

		case swap, ok := <-parser.NewSwapsChan:
			if !ok {
				zap.L().Debug("Swap channel closed")
				return
			}
			if swap == nil {
				continue
			}

			swapBatch = append(swapBatch, *swap)
			zap.L().Debug("new swap received from chan", zap.Any("data", swap))
			if err := database.AddOrUpdatePool(context.Background(), *swap); err != nil {
				zap.L().Error("error adding pools to DB", zap.Error(err))
			} else {
				zap.L().Debug("added pools to DB", zap.Any("data", swap))
			}

			if len(swapBatch) >= 50 {
				if err := database.AddSwapsBatch(context.Background(), swapBatch); err != nil {
					zap.L().Error("error adding swaps batch to DB", zap.Error(err))
					continue
				} else {
					zap.L().Debug("added swaps batch to DB", zap.Int("batch_size", len(swapBatch)))
				}
				swapBatch = swapBatch[:0] // Reset batch
			}

		}
	}
}
