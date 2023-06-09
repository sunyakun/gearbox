package controller

import (
	"github.com/sunyakun/gearbox/pkg/apis"
)

type CreateEvent struct {
	Object apis.Object
}

type UpdateEvent struct {
	ObjectNew apis.Object
}

type DeleteEvent struct {
	Object apis.Object

	// DeleteStateUnknown is true if the Delete event was missed but we identified the object
	// as having been deleted.
	DeleteStateUnknown bool
}

// GenericEvent is an event where the operation type is unknown (e.g. polling or event originating outside the cluster).
// GenericEvent should be generated by a source.Source and transformed into a reconcile.Request by an
// handler.EventHandler.
type GenericEvent struct {
	Object apis.Object
}
