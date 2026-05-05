package sandbox

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestPreferredSignersSplitsRSAAlgorithms(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	signers := preferredSigners(signer)
	if len(signers) != 3 {
		t.Fatalf("expected three RSA signers, got %d", len(signers))
	}
	want := []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA}
	for i, signer := range signers {
		multi, ok := signer.(ssh.MultiAlgorithmSigner)
		if !ok {
			t.Fatalf("signer %d does not implement MultiAlgorithmSigner", i)
		}
		algorithms := multi.Algorithms()
		if len(algorithms) != 1 || algorithms[0] != want[i] {
			t.Fatalf("signer %d algorithms = %v, want [%s]", i, algorithms, want[i])
		}
	}
}
