package domain

import "testing"

func TestRequestFingerprintStable(t *testing.T) {
	fp1, err := RequestFingerprint("wallet_1", "wallet_2", 100, "INR")
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := RequestFingerprint("wallet_1", "wallet_2", 100, "INR")
	if err != nil {
		t.Fatal(err)
	}
	if fp1 != fp2 {
		t.Fatalf("fingerprints differ: %s vs %s", fp1, fp2)
	}
}

func TestRequestFingerprintDiffersOnAmount(t *testing.T) {
	fp1, _ := RequestFingerprint("wallet_1", "wallet_2", 100, "INR")
	fp2, _ := RequestFingerprint("wallet_1", "wallet_2", 101, "INR")
	if fp1 == fp2 {
		t.Fatal("expected different fingerprints for different amounts")
	}
}
