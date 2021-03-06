package voprf

import (
	"fmt"

	"github.com/bytemare/cryptotools/hashtogroup/group"
)

// Server holds the (V)OPRF prover data.
type Server struct {
	privateKey group.Scalar
	publicKey  group.Element
	*oprf
}

func (s *Server) evaluate(blinded group.Element) group.Element {
	return blinded.Mult(s.privateKey)
}

func (s *Server) generateProof(blindedElements, evaluatedElements []group.Element) (proofC, proofS group.Scalar) {
	a0, a1 := s.computeComposites(s.privateKey, s.publicKey, blindedElements, evaluatedElements)

	r := s.group.NewScalar().Random()
	a2 := s.group.Base().Mult(r)
	a3 := a1.Mult(r)

	proofC = s.proofScalar(s.publicKey, a0, a1, a2, a3)
	m := proofC.Mult(s.privateKey)
	proofS = r.Sub(m)

	return proofC, proofS
}

// KeyGen generates and sets a new private/public key pair.
func (s *Server) KeyGen() {
	s.privateKey = s.group.NewScalar().Random()
	s.publicKey = s.group.Base().Mult(s.privateKey)
}

// Evaluate the input with the private key.
func (s *Server) Evaluate(blindedElement []byte) (*Evaluation, error) {
	ev := &evaluation{}
	ev.elements = make([]group.Element, 1)

	b, err := s.group.NewElement().Decode(blindedElement)
	if err != nil {
		return nil, fmt.Errorf("OPRF can't evaluate input : %w", err)
	}

	ev.elements[0] = s.evaluate(b)

	if s.mode == Verifiable {
		c, s := s.generateProof([]group.Element{b}, ev.elements)
		ev.proofC = c
		ev.proofS = s
	}

	return ev.serialize(), nil
}

// EvaluateBatch evaluates the input batch of blindedElements and returns a pointer to the Evaluation. If the server
// was set to be un Verifiable mode, the proof will be included in the Evaluation.
func (s *Server) EvaluateBatch(blindedElements [][]byte) (*Evaluation, error) {
	ev := &evaluation{}
	ev.elements = make([]group.Element, len(blindedElements))

	var blinded []group.Element

	if s.mode == Verifiable {
		blinded = make([]group.Element, len(blindedElements))
	}

	for i, b := range blindedElements {
		b, err := s.group.NewElement().Decode(b)
		if err != nil {
			return nil, fmt.Errorf("OPRF can't evaluate input : %w", err)
		}

		if s.mode == Verifiable {
			blinded[i] = b
		}

		ev.elements[i] = s.evaluate(b)
	}

	if s.mode == Verifiable {
		c, s := s.generateProof(blinded, ev.elements)
		ev.proofC = c
		ev.proofS = s
	}

	return ev.serialize(), nil
}

// FullEvaluate reproduces the full PRF but without the blinding operations, using the client's input.
// This should output the same digest as the client's Finalize() function.
func (s *Server) FullEvaluate(input, info []byte) []byte {
	p := s.group.HashToGroup(input)
	t := s.evaluate(p)

	return s.hashTranscript(input, t.Bytes(), info)
}

// VerifyFinalize takes the client input (the un-blinded element) and the client's finalize() output,
// and returns whether it can match the client's output.
func (s *Server) VerifyFinalize(input, output, info []byte) bool {
	digest := s.FullEvaluate(input, info)
	return ctEqual(digest, output)
}

// VerifyFinalizeBatch takes the batch of client input (the un-blinded elements) and the client's finalize() outputs,
// and returns whether it can match the client's outputs.
func (s *Server) VerifyFinalizeBatch(input, output [][]byte, info []byte) bool {
	res := true

	for i, in := range input {
		digest := s.FullEvaluate(in, info)
		res = res && ctEqual(digest, output[i])
	}

	return res
}

// PrivateKey returns the server's serialized private key.
func (s *Server) PrivateKey() []byte {
	return s.privateKey.Bytes()
}

// PublicKey returns the server's serialized public key.
func (s *Server) PublicKey() []byte {
	return s.publicKey.Bytes()
}
