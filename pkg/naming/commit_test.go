package naming

import (
	"testing"
	"time"
)

func TestCommitReveal(t *testing.T) {
	name := "example.mskt"
	owner := "did:mskt:testowner123456"
	secret, err := GenerateSecret()
	if err != nil {
		t.Fatal(err)
	}

	hash, err := CommitmentHash(name, owner, secret)
	if err != nil {
		t.Fatal(err)
	}

	commit, err := NewDomainCommitRecord(name, owner, secret)
	if err != nil {
		t.Fatal(err)
	}
	if commit.Commitment != hash {
		t.Fatal("commitment hash mismatch")
	}

	// قبل انتهاء فترة الانتظار — يجب أن يفشل
	if err := VerifyReveal(commit, name, owner, secret); err == nil {
		t.Fatal("should fail before reveal delay")
	}

	// محاكاة انتهاء الانتظار
	commit.CommittedAt = time.Now().Add(-MinRevealDelay - time.Second).Unix()
	if err := VerifyReveal(commit, name, owner, secret); err != nil {
		t.Fatalf("reveal should succeed: %v", err)
	}

	// سر خاطئ
	if err := VerifyReveal(commit, name, owner, "wrong-secret"); err == nil {
		t.Fatal("wrong secret should fail")
	}
}
