package apis

import "time"

const (
	StatusFailure = "Failure"
	StatusSuccess = "Success"
)

type Object interface {
	GetKey() string
	SetKey(string)
	GetKind() string
	SetKind(string)
	GetResourceVersion() string
}

type ObjectMeta struct {
	Kind            string    `json:"kind,omitempty"`
	Key             string    `json:"key,omitempty"`
	ResourceVersion string    `json:"resourceVersion,omitempty"`
	CreateTime      time.Time `json:"createTime,omitempty"`
	UpdateTime      time.Time `json:"updateTime,omitempty"`
}

func (o *ObjectMeta) GetKey() string {
	return o.Key
}

func (o *ObjectMeta) SetKey(key string) {
	o.Key = key
}

func (o *ObjectMeta) GetKind() string {
	return o.Kind
}

func (o *ObjectMeta) SetKind(kind string) {
	o.Kind = kind
}

func (o *ObjectMeta) GetResourceVersion() string {
	return o.ResourceVersion
}

type ListOptions struct {
	Limit    int    `json:"limit,omitempty" query:"limit"`
	Offset   int    `json:"offset,omitempty" query:"offset"`
	Selector string `json:"selector,omitempty" query:"selector"`
}

type ObjectList[T Object] struct {
	Count    int64 `json:"count"`
	Continue bool  `json:"continue"`
	Items    []T   `json:"items"`
}

type Status struct {
	ObjectMeta `json:"metadata,omitempty"`
	Code       int `json:"code"`
	// Status of the the operation
	Status string `json:"status"`
	// Reason is the machine-readable description for current status
	Reason string `json:"reason"`
	// Message is the human-readable description for current status
	Message string `json:"message"`
}
