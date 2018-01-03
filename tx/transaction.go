package tx

import (
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/vechain/thor/acc"
	"github.com/vechain/thor/bn"
	"github.com/vechain/thor/cry"
	"github.com/vechain/thor/dsa"
)

// Transaction is an immutable tx type.
type Transaction struct {
	body body

	cache struct {
		hash   *cry.Hash
		signer *acc.Address
	}
}

// body describes details of a tx.
type body struct {
	Clauses     Clauses
	GasPrice    bn.Int
	GasLimit    bn.Int
	Nonce       uint64
	TimeBarrier uint64
	DependsOn   *cry.Hash `rlp:"nil"`
	Signature   []byte
}

// Hash returns hash of tx.
func (t *Transaction) Hash() cry.Hash {
	if cached := t.cache.hash; cached != nil {
		return *cached
	}

	hw := cry.NewHasher()
	rlp.Encode(hw, t)

	var h cry.Hash
	hw.Sum(h[:0])
	t.cache.hash = &h
	return h
}

// HashOfWorkProof returns hash for work proof.
func (t *Transaction) HashOfWorkProof() (hash cry.Hash) {
	hw := cry.NewHasher()
	rlp.Encode(hw, []interface{}{
		t.body.Clauses,
		t.body.GasPrice,
		t.body.GasLimit,
		t.body.Nonce,
		t.body.DependsOn,
	})
	hw.Sum(hash[:0])
	return
}

// HashForSigning returns hash of tx excludes signature.
func (t *Transaction) HashForSigning() cry.Hash {
	wph := t.HashOfWorkProof()
	// use hash of work proof hash as signing hash
	return cry.HashSum(wph[:])
}

// GasPrice returns gas price.
func (t *Transaction) GasPrice() bn.Int {
	return t.body.GasPrice
}

// GasLimit returns gas limit.
func (t *Transaction) GasLimit() bn.Int {
	return t.body.GasLimit
}

// TimeBarrier returns time barrier.
// It's required that tx.TimeBarrier <= block.Timestamp,
// when a tx was packed in a block.
func (t *Transaction) TimeBarrier() uint64 {
	return t.body.TimeBarrier
}

// Clauses returns caluses in tx.
func (t *Transaction) Clauses() Clauses {
	clauses := make(Clauses, len(t.body.Clauses))
	for i, c := range t.body.Clauses {
		clauses[i] = c.Copy()
	}
	return clauses
}

// WithSignature create a new tx with signature set.
func (t *Transaction) WithSignature(sig []byte) *Transaction {
	newTx := Transaction{
		body: t.body,
	}
	// copy sig
	newTx.body.Signature = append([]byte(nil), sig...)
	return &newTx
}

// Signer returns the signer of tx.
func (t *Transaction) Signer() (*acc.Address, error) {
	if len(t.body.Signature) == 0 {
		return nil, errors.New("not signed")
	}
	if signer := t.cache.signer; signer != nil {
		cpy := *signer
		return &cpy, nil
	}
	signer, err := dsa.Signer(t.HashForSigning(), t.body.Signature)
	if err != nil {
		return nil, err
	}
	t.cache.signer = signer
	cpy := *signer
	return &cpy, nil
}

// EncodeRLP implements rlp.Encoder
func (t *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &t.body)
}

// DecodeRLP implements rlp.Decoder
func (t *Transaction) DecodeRLP(s *rlp.Stream) error {
	var body body
	if err := s.Decode(&body); err != nil {
		return err
	}
	*t = Transaction{
		body: body,
	}
	return nil
}
