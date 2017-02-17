package renter

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"testing"

	"github.com/NebulousLabs/Sia/crypto"
)

// TestRSEncode tests the rsCode type.
func TestRSEncode(t *testing.T) {
	badParams := []struct {
		data, parity int
	}{
		{-1, -1},
		{-1, 0},
		{0, -1},
		{0, 0},
		{0, 1},
		{1, 0},
	}
	for _, ps := range badParams {
		if _, err := NewRSCode(ps.data, ps.parity); err == nil {
			t.Error("expected bad parameter error, got nil")
		}
	}

	rsc, err := NewRSCode(10, 3)
	if err != nil {
		t.Fatal(err)
	}

	data := make([]byte, 777)
	rand.Read(data)

	pieces, err := rsc.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rsc.Encode(nil)
	if err == nil {
		t.Fatal("expected nil data error, got nil")
	}

	buf := new(bytes.Buffer)
	err = rsc.Recover(pieces, 777, buf)
	if err != nil {
		t.Fatal(err)
	}
	err = rsc.Recover(nil, 777, buf)
	if err == nil {
		t.Fatal("expected nil pieces error, got nil")
	}

	if !bytes.Equal(data, buf.Bytes()) {
		t.Fatal("recovered data does not match original")
	}
}

func TestRSFragmentEquivalent(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	// Set the erasure coding parameters and create the reference data.
	dataN := 10
	parityN := 20
	pieceSize := 1 << 22
	fragmentSize := 1 << 21
	original := make([]byte, pieceSize*dataN)
	rand.Read(original)

	// Erasure code the data.
	rsc, err := NewRSCode(dataN, parityN)
	if err != nil {
		t.Fatal(err)
	}
	src1 := make([]byte, pieceSize*dataN)
	copy(src1, original)
	pieces, err := rsc.Encode(src1)
	if err != nil {
		t.Fatal(err)
	}

	// Destroy a bunch of pieces.
	perm, err := crypto.Perm(dataN + parityN)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < parityN; i++ {
		pieces[perm[i]] = nil
	}

	// Build out the fragment set.
	fragmentSet := make([][]byte, dataN+parityN)
	// Recover the pieces using tiny fragments of size 1<<12.
	recovered := make([]byte, pieceSize*dataN)
	for i := 0; i < (pieceSize / fragmentSize); i++ {
		// Assemble the fragments.
		for j := 0; j < dataN+parityN; j++ {
			if pieces[j] != nil {
				fragmentSet[j] = pieces[j][i*fragmentSize : (i+1)*fragmentSize]
			}
		}

		// Recover the fragments.
		buf := new(bytes.Buffer)
		err = rsc.Recover(fragmentSet, uint64(fragmentSize*dataN), buf)
		if err != nil {
			t.Fatal(err)
		}

		// Copy the recovered bits into the final recovered slice.
		recoveredBits := buf.Bytes()
		copy(recovered[i*fragmentSize*dataN:(i+1)*fragmentSize*dataN], recoveredBits)
	}

	// Verify that recovery was correct.
	if !bytes.Equal(recovered, original) {
		t.Error("data mismatch")
	}
}

func BenchmarkRSEncode(b *testing.B) {
	rsc, err := NewRSCode(80, 20)
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, 1<<20)
	rand.Read(data)

	b.SetBytes(1 << 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsc.Encode(data)
	}
}

func BenchmarkRSRecover(b *testing.B) {
	rsc, err := NewRSCode(50, 200)
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, 1<<20)
	rand.Read(data)
	pieces, err := rsc.Encode(data)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(1 << 20)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pieces[0] = nil
		rsc.Recover(pieces, 1<<20, ioutil.Discard)
	}
}
