package admission

import (
	"context"

	"github.com/sunyakun/gearbox/pkg/apis"
)

// Operation is the type of resource operation being checked for admission control
type Operation string

// Operation constants
const (
	Create  Operation = "CREATE"
	Update  Operation = "UPDATE"
	Delete  Operation = "DELETE"
	Connect Operation = "CONNECT"
)

// Attributes is an interface used by AdmissionController to get information about a request
// that is used to make an admission decision.
type Attributes interface {
	// GetOperation is the operation being performed
	GetOperation() Operation
	// GetObject is the object from the incoming request
	GetObject() apis.Object
	// GetResource is the resource name
	GetResource() string
}

// Interface is an abstract, pluggable interface for Admission Control decisions.
type Interface interface {
	// Handles returns true if this admission controller can handle the given operation
	// where operation can be one of CREATE, UPDATE, DELETE, or CONNECT
	Handles(operation Operation) bool
}

type MutationInterface interface {
	Interface

	// Admit makes an admission decision based on the request attributes.
	// Context is used only for timeout/deadline/cancellation and tracing information.
	Admit(ctx context.Context, a Attributes) (err error)
}

// ValidationInterface is an abstract, pluggable interface for Admission Control decisions.
type ValidationInterface interface {
	Interface

	// Validate makes an admission decision based on the request attributes.  It is NOT allowed to mutate
	// Context is used only for timeout/deadline/cancellation and tracing information.
	Validate(ctx context.Context, a Attributes) (err error)
}
