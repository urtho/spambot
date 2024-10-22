package worker

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/algorand/go-algorand-sdk/v2/crypto"
	"github.com/algorand/go-algorand-sdk/v2/transaction"
	"github.com/algorand/go-algorand-sdk/v2/types"
	"go.uber.org/ratelimit"
)

func (w *SPAMWorker) createCleanupSTXGrp(receiverAddress string) (Stx, error) {
	var params *types.SuggestedParams
	w.sParams.RLock()
	params = w.sParams.params
	w.sParams.RUnlock()

	totalFee := types.MicroAlgos(2000)

	txn1, err := transaction.MakePaymentTxn(
		w.spamAccount.Address.String(),
		w.spamAccount.Address.String(),
		0,
		nil, // note
		"",  // close remainder to
		*params,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating first transaction: %v", err)
	}
	// Set the pooled fee
	txn1.Fee = totalFee

	sender := crypto.GenerateAccount()

	txn2, err := transaction.MakePaymentTxn(
		sender.Address.String(),
		receiverAddress,
		0,
		nil, // note
		"",  // close remainder to
		*params,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating second transaction: %v", err)
	}
	txn2.Fee = 0

	// Group transactions
	gid, err := crypto.ComputeGroupID([]types.Transaction{txn1, txn2})
	if err != nil {
		return nil, fmt.Errorf("error computing group ID: %v", err)
	}

	// Assign group ID to both transactions
	txn1.Group = gid
	txn2.Group = gid

	// Sign transactions
	_, signedTxn1, err := crypto.SignTransaction(w.spamAccount.PrivateKey, txn1)
	if err != nil {
		return nil, fmt.Errorf("error signing first transaction: %v", err)
	}

	_, signedTxn2, err := crypto.SignTransaction(sender.PrivateKey, txn2)
	if err != nil {
		return nil, fmt.Errorf("error signing second transaction: %v", err)
	}

	// Combine signed transactions
	var signedGroup []byte
	signedGroup = append(signedGroup, signedTxn1...)
	signedGroup = append(signedGroup, signedTxn2...)

	return signedGroup, nil
}

func (w *SPAMWorker) spamCleanFile(ctx context.Context) error {

	file, err := os.Open(w.cfg.SPAM.CleanFile)
	if err != nil {
		w.log.Errorf("Error opening file: %v", err)
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	rl := ratelimit.New(w.cfg.SPAM.Rate, ratelimit.WithoutSlack)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}

		rl.Take()
		addr := scanner.Text()
		stx, err := w.createCleanupSTXGrp(addr)
		if err != nil {
			w.log.Errorf("Error creating STX: %v", err)
		}
		w.txChan <- stx
	}

	if err := scanner.Err(); err != nil {
		w.log.Errorf("Error reading file: %v", err)
	}

	return nil

}
