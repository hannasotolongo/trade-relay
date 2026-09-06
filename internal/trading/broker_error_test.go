package trading

import (
	"errors"
	"testing"
)

func TestSubmissionOutcomeFromError(t *testing.T) {
	baseErr := errors.New("network timeout")

	err := NewBrokerSubmissionError(
		SubmissionAmbiguous,
		baseErr,
	)

	outcome, ok := SubmissionOutcomeFromError(err)
	if !ok {
		t.Fatal("expected broker submission error")
	}

	if outcome != SubmissionAmbiguous {
		t.Fatalf(
			"expected outcome %s, got %s",
			SubmissionAmbiguous,
			outcome,
		)
	}

	if !errors.Is(err, baseErr) {
		t.Fatal("expected wrapped error to be discoverable with errors.Is")
	}
}

func TestSubmissionOutcomeFromErrorReturnsFalseForRegularError(t *testing.T) {
	err := errors.New("regular error")

	outcome, ok := SubmissionOutcomeFromError(err)

	if ok {
		t.Fatal("expected regular error not to be classified")
	}

	if outcome != "" {
		t.Fatalf("expected empty outcome, got %s", outcome)
	}
}

func TestBrokerSubmissionErrorWithoutUnderlyingError(t *testing.T) {
	err := NewBrokerSubmissionError(
		SubmissionNotSent,
		nil,
	)

	if err.Error() != "broker submission failed" {
		t.Fatalf(
			"expected fallback error message, got %q",
			err.Error(),
		)
	}

	outcome, ok := SubmissionOutcomeFromError(err)
	if !ok {
		t.Fatal("expected broker submission error")
	}

	if outcome != SubmissionNotSent {
		t.Fatalf(
			"expected outcome %s, got %s",
			SubmissionNotSent,
			outcome,
		)
	}
}
