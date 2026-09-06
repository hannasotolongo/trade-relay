package trading

import "errors"

type SubmissionOutcome string

const (
	SubmissionNotSent   SubmissionOutcome = "NOT_SENT"
	SubmissionAmbiguous SubmissionOutcome = "AMBIGUOUS"
)

type BrokerSubmissionError struct {
	Outcome SubmissionOutcome
	Err     error
}

func (e *BrokerSubmissionError) Error() string {
	if e.Err == nil {
		return "broker submission failed"
	}

	return e.Err.Error()
}

func (e *BrokerSubmissionError) Unwrap() error {
	return e.Err
}

func NewBrokerSubmissionError(
	outcome SubmissionOutcome,
	err error,
) error {
	return &BrokerSubmissionError{
		Outcome: outcome,
		Err:     err,
	}
}

func SubmissionOutcomeFromError(err error) (SubmissionOutcome, bool) {
	var submissionErr *BrokerSubmissionError

	if !errors.As(err, &submissionErr) {
		return "", false
	}

	return submissionErr.Outcome, true
}
