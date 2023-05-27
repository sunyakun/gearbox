package apis

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Foo struct {
	Bar       string       `json:"bar,omitempty"`
	Extension RawExtension `json:"extension,omitempty"`
}

func TestRawExtension(t *testing.T) {
	bs, err := json.Marshal(Foo{
		Bar:       "bar",
		Extension: RawExtension{},
	})
	assert.Nil(t, err)
	assert.Equal(t, bs, []byte(`{"bar":"bar","extension":null}`))

	bs, err = json.Marshal(Foo{
		Bar:       "bar",
		Extension: RawExtension{Raw: []byte(`{"bar":"foo"}`)},
	})
	assert.Nil(t, err)
	assert.Equal(t, bs, []byte(`{"bar":"bar","extension":{"bar":"foo"}}`))

	var f Foo
	err = json.Unmarshal([]byte(`{"bar":"bar","extension":{"bar":"foo"}}`), &f)
	assert.Nil(t, err)
	assert.Equal(t, f.Bar, "bar")
	assert.Equal(t, f.Extension.Raw, []byte(`{"bar":"foo"}`))

	err = json.Unmarshal(f.Extension.Raw, &f)
	assert.Nil(t, err)
	assert.Equal(t, f.Bar, "foo")
}
