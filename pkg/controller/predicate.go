package controller

// Predicate filters events before enqueuing the keys.
type Predicate interface {
	// Create returns true if the Create event should be processed
	Create(CreateEvent) bool

	// Delete returns true if the Delete event should be processed
	Delete(DeleteEvent) bool

	// Update returns true if the Update event should be processed
	Update(UpdateEvent) bool

	// Generic returns true if the Generic event should be processed
	Generic(GenericEvent) bool
}

type PredicateFunc struct {
	CreateFunc  func(CreateEvent) bool
	DeleteFunc  func(DeleteEvent) bool
	UpdateFunc  func(UpdateEvent) bool
	GenericFunc func(GenericEvent) bool
}

func (p PredicateFunc) Create(evt CreateEvent) bool {
	if p.CreateFunc != nil {
		return p.CreateFunc(evt)
	}
	return false
}

func (p PredicateFunc) Delete(evt DeleteEvent) bool {
	if p.DeleteFunc != nil {
		return p.DeleteFunc(evt)
	}
	return false
}

func (p PredicateFunc) Update(evt UpdateEvent) bool {
	if p.UpdateFunc != nil {
		return p.UpdateFunc(evt)
	}
	return false
}

func (p PredicateFunc) Generic(evt GenericEvent) bool {
	if p.GenericFunc != nil {
		return p.GenericFunc(evt)
	}
	return false
}
