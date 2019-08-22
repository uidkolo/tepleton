package crypto

import (
	"bytes"
	"fmt"

	. "github.com/tepleton/go-common"
	data "github.com/tepleton/go-data"
	"github.com/tepleton/go-wire"
)

// SignatureInner is now the interface itself.
// Use Signature in all code
type SignatureInner interface {
	Bytes() []byte
	IsZero() bool
	String() string
	Equals(Signature) bool
}

var sigMapper data.Mapper

// register both public key types with go-data (and thus go-wire)
func init() {
	sigMapper = data.NewMapper(Signature{}).
		RegisterImplementation(SignatureEd25519{}, NameEd25519, TypeEd25519).
		RegisterImplementation(SignatureSecp256k1{}, NameSecp256k1, TypeSecp256k1)
}

// Signature add json serialization to Signature
type Signature struct {
	SignatureInner
}

func WrapSignature(pk SignatureInner) Signature {
	if wrap, ok := pk.(Signature); ok {
		pk = wrap.Unwrap()
	}
	return Signature{pk}
}

func (p Signature) Unwrap() SignatureInner {
	pk := p.SignatureInner
	for wrap, ok := pk.(Signature); ok; wrap, ok = pk.(Signature) {
		pk = wrap.SignatureInner
	}
	return pk
}

func (p Signature) MarshalJSON() ([]byte, error) {
	return sigMapper.ToJSON(p.SignatureInner)
}

func (p *Signature) UnmarshalJSON(data []byte) (err error) {
	parsed, err := sigMapper.FromJSON(data)
	if err == nil && parsed != nil {
		p.SignatureInner = parsed.(SignatureInner)
	}
	return
}

func (p Signature) Empty() bool {
	return p.SignatureInner == nil
}

func SignatureFromBytes(sigBytes []byte) (sig Signature, err error) {
	err = wire.ReadBinaryBytes(sigBytes, &sig)
	return
}

//-------------------------------------

// Implements Signature
type SignatureEd25519 [64]byte

func (sig SignatureEd25519) Bytes() []byte {
	return wire.BinaryBytes(Signature{sig})
}

func (sig SignatureEd25519) IsZero() bool { return len(sig) == 0 }

func (sig SignatureEd25519) String() string { return fmt.Sprintf("/%X.../", Fingerprint(sig[:])) }

func (sig SignatureEd25519) Equals(other Signature) bool {
	if otherEd, ok := other.Unwrap().(SignatureEd25519); ok {
		return bytes.Equal(sig[:], otherEd[:])
	} else {
		return false
	}
}

func (p SignatureEd25519) MarshalJSON() ([]byte, error) {
	return data.Encoder.Marshal(p[:])
}

func (p *SignatureEd25519) UnmarshalJSON(enc []byte) error {
	var ref []byte
	err := data.Encoder.Unmarshal(&ref, enc)
	copy(p[:], ref)
	return err
}

//-------------------------------------

// Implements Signature
type SignatureSecp256k1 []byte

func (sig SignatureSecp256k1) Bytes() []byte {
	return wire.BinaryBytes(Signature{sig})
}

func (sig SignatureSecp256k1) IsZero() bool { return len(sig) == 0 }

func (sig SignatureSecp256k1) String() string { return fmt.Sprintf("/%X.../", Fingerprint(sig[:])) }

func (sig SignatureSecp256k1) Equals(other Signature) bool {
	if otherEd, ok := other.Unwrap().(SignatureSecp256k1); ok {
		return bytes.Equal(sig[:], otherEd[:])
	} else {
		return false
	}
}
func (p SignatureSecp256k1) MarshalJSON() ([]byte, error) {
	return data.Encoder.Marshal(p)
}

func (p *SignatureSecp256k1) UnmarshalJSON(enc []byte) error {
	return data.Encoder.Unmarshal((*[]byte)(p), enc)
}
