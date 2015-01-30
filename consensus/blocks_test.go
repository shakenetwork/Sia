package consensus

import (
	"testing"
	"time"
)

// mineTestingBlock accepts a bunch of parameters for a block and then grinds
// blocks until a block with the appropriate target is found.
func mineTestingBlock(parent BlockID, timestamp Timestamp, minerAddress CoinAddress, txns []Transaction, target Target) (b Block, err error) {
	if RootTarget[0] != 64 {
		panic("using wrong constant during testing!")
	}

	b = Block{
		ParentBlockID: parent,
		Timestamp:     timestamp,
		MinerAddress:  minerAddress,
		Transactions:  txns,
	}

	for !b.CheckTarget(target) && b.Nonce < 1000*1000*1000 {
		b.Nonce++
	}
	if !b.CheckTarget(target) {
		panic("mineTestingBlock failed!")
	}
	return
}

// mineValidBlock picks valid/legal parameters for a block and then uses them
// to call mineTestingBlock.
func mineValidBlock(s *State) (b Block, err error) {
	return mineTestingBlock(s.CurrentBlock().ID(), Timestamp(time.Now().Unix()), CoinAddress{}, nil, s.CurrentTarget())
}

// testEmptyBlock adds an empty block to the state and checks for errors.
func testEmptyBlock(t *testing.T, s *State) {
	// Get prior stats about the state.
	bbLen := len(s.badBlocks)
	bmLen := len(s.blockMap)
	mpLen := len(s.missingParents)
	cpLen := len(s.currentPath)
	uoLen := len(s.unspentOutputs)
	ocLen := len(s.openContracts)

	// Mine and submit a block
	b, err := mineValidBlock(s)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AcceptBlock(b)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the state has updated as expected:
	//		bad blocks should not change
	//		blockMap should get 1 new member
	//		missingParents should not change
	//		currentPath should get 1 new member
	//		unspentOutputs should grow by at least 1 (missedProofs can make it grow by more)
	//		openContracts should not grow (contracts may close during the block though)
	if bbLen != len(s.badBlocks) ||
		bmLen != len(s.blockMap)-1 ||
		mpLen != len(s.missingParents) ||
		cpLen != len(s.currentPath)-1 ||
		uoLen > len(s.unspentOutputs)-1 ||
		ocLen < len(s.openContracts) {
		t.Error("state changed unexpectedly after accepting an empty block")
	}
	if s.currentBlockID != b.ID() {
		t.Error("the state's current block id did not change after getting a new block")
	}
	if s.currentPath[s.Height()] != b.ID() {
		t.Error("the state's current path didn't update correctly after accepting a new block")
	}
	_, exists := s.blockMap[b.ID()]
	if !exists {
		t.Error("the state's block map did not update correctly after getting an empty block")
	}
	_, exists = s.unspentOutputs[b.SubsidyID()]
	if !exists {
		t.Error("the blocks subsidy output did not get added to the set of unspent outputs")
	}
}

// testLargeBlock creates a block that is too large to be accepted by the state
// and checks that it actually gets rejected.
func testLargeBlock(t *testing.T, s *State) {
	txns := make([]Transaction, 1)
	bigData := string(make([]byte, BlockSizeLimit)) // TODO: test all the way down to one byte over the limit.
	txns[0] = Transaction{
		ArbitraryData: []string{bigData},
	}
	b, err := mineTestingBlock(s.CurrentBlock().ID(), Timestamp(time.Now().Unix()), CoinAddress{}, txns, s.CurrentTarget())
	if err != nil {
		t.Fatal(err)
	}

	err = s.AcceptBlock(b)
	if err != LargeBlockErr {
		t.Fatal(err)
	}
}

// testRepeatBlock submits a block to the state, and then submits the same
// block to the state. If anything in the state has changed, an error is noted.
func testRepeatBlock(t *testing.T, s *State) {
	// Add a non-repeat block to the state.
	b, err := mineValidBlock(s)
	if err != nil {
		t.Fatal(err)
	}
	err = s.AcceptBlock(b)
	if err != nil {
		t.Fatal(err)
	}

	// Collect metrics about the state.
	bbLen := len(s.badBlocks)
	bmLen := len(s.blockMap)
	mpLen := len(s.missingParents)
	cpLen := len(s.currentPath)
	uoLen := len(s.unspentOutputs)
	ocLen := len(s.openContracts)
	stateHash := s.StateHash()

	// Submit the repeat block.
	err = s.AcceptBlock(b)
	if err != BlockKnownErr {
		t.Error("expecting BlockKnownErr, got", err)
	}

	// Compare the metrics and report an error if something has changed.
	if bbLen != len(s.badBlocks) ||
		bmLen != len(s.blockMap) ||
		mpLen != len(s.missingParents) ||
		cpLen != len(s.currentPath) ||
		uoLen != len(s.unspentOutputs) ||
		ocLen != len(s.openContracts) ||
		stateHash != s.StateHash() {
		t.Error("state changed after getting a repeat block.")
	}
}

// TestEmptyBlock creates a new state and uses it to call testEmptyBlock.
func TestEmptyBlock(t *testing.T) {
	s := CreateGenesisState()
	testEmptyBlock(t, s)
}

// TestLargeBlock creates a new state and uses it to call testLargeBlock.
func TestLargeBlock(t *testing.T) {
	s := CreateGenesisState()
	testLargeBlock(t, s)
}

// TestRepeatBlock creates a new state and uses it to call testRepeatBlock.
func TestRepeatBlock(t *testing.T) {
	s := CreateGenesisState()
	testRepeatBlock(t, s)
}
