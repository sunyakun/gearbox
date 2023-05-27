package admission

import "context"

// chainAdmissionHandler is an instance of admission.NamedHandler that performs admission control using
// a chain of admission handlers
type chainAdmissionHandler []Interface

// NewChainHandler creates a new chain handler from an array of handlers. Used for testing.
func NewChainHandler(handlers ...Interface) chainAdmissionHandler {
	return chainAdmissionHandler(handlers)
}

// Admit performs an admission control check using a chain of handlers, and returns immediately on first error
func (admissionHandler chainAdmissionHandler) Admit(ctx context.Context, a Attributes) error {
	for _, handler := range admissionHandler {
		if !handler.Handles(a.GetOperation()) {
			continue
		}
		if mutator, ok := handler.(MutationInterface); ok {
			err := mutator.Admit(ctx, a)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Validate performs an admission control check using a chain of handlers, and returns immediately on first error
func (admissionHandler chainAdmissionHandler) Validate(ctx context.Context, a Attributes) error {
	for _, handler := range admissionHandler {
		if !handler.Handles(a.GetOperation()) {
			continue
		}
		if validator, ok := handler.(ValidationInterface); ok {
			err := validator.Validate(ctx, a)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Handles will return true if any of the handlers handles the given operation
func (admissionHandler chainAdmissionHandler) Handles(operation Operation) bool {
	for _, handler := range admissionHandler {
		if handler.Handles(operation) {
			return true
		}
	}
	return false
}
