package transactionpool

import (
	"testing"

	"github.com/NebulousLabs/Sia/modules"
	"github.com/NebulousLabs/Sia/types"

	"github.com/NebulousLabs/fastrand"
)

// BenchmarkAccept500ArbTransactions tracks the amount of time that it takes to
// add a transaction to the transaction pool.
func BenchmarkAccept500ArbTransactions(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}

	// Run the test to, each run, create a completely fresh tpool tester and add
	// about 500 arbitrary data transactions to it.
	for i := 0; i < b.N; i++ {
		tpt, err := createTpoolTester(b.Name())
		if err != nil {
			b.Fatal(err)
		}
		for j := 0; j < 500; j++ {
			data := fastrand.Bytes(16) // Small amounts of data.
			data = append(modules.PrefixNonSia[:], data...)
			txn := types.Transaction{
				ArbitraryData: [][]byte{data},
			}
			err := tpt.tpool.AcceptTransactionSet([]types.Transaction{txn})
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkAccept1000ArbTransactions tracks the amount of time that it takes to
// add a transaction to the transaction pool.
func BenchmarkAccept1000ArbTransactions(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}

	// Run the test to, each run, create a completely fresh tpool tester and add
	// about 1000 arbitrary data transactions to it.
	for i := 0; i < b.N; i++ {
		tpt, err := createTpoolTester(b.Name())
		if err != nil {
			b.Fatal(err)
		}
		for j := 0; j < 1000; j++ {
			data := fastrand.Bytes(16) // Small amounts of data.
			data = append(modules.PrefixNonSia[:], data...)
			txn := types.Transaction{
				ArbitraryData: [][]byte{data},
			}
			err := tpt.tpool.AcceptTransactionSet([]types.Transaction{txn})
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkAccept2000ArbTransactions tracks the amount of time that it takes to
// add a transaction to the transaction pool.
func BenchmarkAccept2000ArbTransactions(b *testing.B) {
	if testing.Short() {
		b.SkipNow()
	}

	// Run the test to, each run, create a completely fresh tpool tester and add
	// about 2000 arbitrary data transactions to it.
	for i := 0; i < b.N; i++ {
		tpt, err := createTpoolTester(b.Name())
		if err != nil {
			b.Fatal(err)
		}
		for j := 0; j < 2000; j++ {
			data := fastrand.Bytes(16) // Small amounts of data.
			data = append(modules.PrefixNonSia[:], data...)
			txn := types.Transaction{
				ArbitraryData: [][]byte{data},
			}
			err := tpt.tpool.AcceptTransactionSet([]types.Transaction{txn})
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
