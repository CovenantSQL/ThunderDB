/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package types

import (
	"time"

	"github.com/CovenantSQL/CovenantSQL/crypto"
	ca "github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/hash"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/merkle"
	"github.com/CovenantSQL/CovenantSQL/proto"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

//go:generate hsp

// Header is a block header.
type Header struct {
	Version     int32
	Producer    proto.NodeID
	GenesisHash hash.Hash
	ParentHash  hash.Hash
	MerkleRoot  hash.Hash
	Timestamp   time.Time
}

// SignedHeader is block header along with its producer signature.
type SignedHeader struct {
	Header
	HSV crypto.DefaultHashSignVerifierImpl
}

func (s *SignedHeader) Sign(signer *ca.PrivateKey) error {
	return s.HSV.Sign(&s.Header, signer)
}

// Verify verifies the signature of the signed header.
func (s *SignedHeader) Verify() error {
	return s.HSV.Verify(&s.Header)
}

// VerifyAsGenesis verifies the signed header as a genesis block header.
func (s *SignedHeader) VerifyAsGenesis() (err error) {
	var pk *ca.PublicKey
	log.WithFields(log.Fields{
		"producer": s.Producer,
		"root":     s.GenesisHash.String(),
		"parent":   s.ParentHash.String(),
		"merkle":   s.MerkleRoot.String(),
		"block":    s.HSV.Hash().String(),
	}).Debug("Verifying genesis block header")
	if pk, err = kms.GetPublicKey(s.Producer); err != nil {
		return
	}
	if !pk.IsEqual(s.HSV.Signee) {
		err = ErrNodePublicKeyNotMatch
		return
	}
	return s.Verify()
}

// Block is a node of blockchain.
type Block struct {
	SignedHeader SignedHeader
	Queries      []*hash.Hash
}

// PackAndSignBlock generates the signature for the Block from the given PrivateKey.
func (b *Block) PackAndSignBlock(signer *ca.PrivateKey) (err error) {
	// Calculate merkle root
	b.SignedHeader.MerkleRoot = *merkle.NewMerkle(b.Queries).GetRoot()
	return b.SignedHeader.Sign(signer)
}

// PushAckedQuery pushes a acknowledged and verified query into the block.
func (b *Block) PushAckedQuery(h *hash.Hash) {
	b.Queries = append(b.Queries, h)
}

// Verify verifies the merkle root and header signature of the block.
func (b *Block) Verify() (err error) {
	// Verify merkle root
	if MerkleRoot := *merkle.NewMerkle(b.Queries).GetRoot(); !MerkleRoot.IsEqual(
		&b.SignedHeader.MerkleRoot,
	) {
		return ErrMerkleRootVerification
	}
	return b.SignedHeader.Verify()
}

// VerifyAsGenesis verifies the block as a genesis block.
func (b *Block) VerifyAsGenesis() (err error) {
	var pk *ca.PublicKey
	if pk, err = kms.GetPublicKey(b.SignedHeader.Producer); err != nil {
		return
	}
	if !pk.IsEqual(b.SignedHeader.HSV.Signee) {
		err = ErrNodePublicKeyNotMatch
		return
	}
	return b.Verify()
}

// Timestamp returns the timestamp field of the block header.
func (b *Block) Timestamp() time.Time {
	return b.SignedHeader.Timestamp
}

// Producer returns the producer field of the block header.
func (b *Block) Producer() proto.NodeID {
	return b.SignedHeader.Producer
}

// ParentHash returns the parent hash field of the block header.
func (b *Block) ParentHash() *hash.Hash {
	return &b.SignedHeader.ParentHash
}

// BlockHash returns the parent hash field of the block header.
func (b *Block) BlockHash() *hash.Hash {
	return &b.SignedHeader.HSV.DataHash
}

// GenesisHash returns the parent hash field of the block header.
func (b *Block) GenesisHash() *hash.Hash {
	return &b.SignedHeader.GenesisHash
}

// Signee returns the signee field  of the block signed header.
func (b *Block) Signee() *ca.PublicKey {
	return b.SignedHeader.HSV.Signee
}

// Blocks is Block (reference) array.
type Blocks []*Block