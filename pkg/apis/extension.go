package apis

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type RawExtension struct {
	Object any    `json:"-"`
	Raw    []byte `json:"-"`
}

func (re *RawExtension) UnmarshalJSON(b []byte) error {
	if re == nil {
		return fmt.Errorf("apis.RawExtension: unmarshal json to nil")
	}
	if !bytes.Equal(b, []byte("null")) {
		re.Raw = append(re.Raw[0:0], b...)
	}
	return nil
}

func (re RawExtension) MarshalJSON() ([]byte, error) {
	var err error
	if re.Raw == nil && re.Object == nil {
		return []byte("null"), nil
	} else if re.Raw == nil && re.Object != nil {
		re.Raw, err = json.Marshal(re.Object)
		if err != nil {
			return nil, err
		}
	}
	return re.Raw, nil
}
