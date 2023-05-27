package admission

import "github.com/sunyakun/gearbox/pkg/apis"

type Attribute struct {
	Object       apis.Object
	Operation    Operation
	ResourceName string
}

func (a *Attribute) GetObject() apis.Object {
	return a.Object
}

func (a *Attribute) GetOperation() Operation {
	return a.Operation
}

func (a *Attribute) GetResource() string {
	return a.ResourceName
}
