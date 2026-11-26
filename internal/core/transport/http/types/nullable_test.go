package types

import (
	"encoding/json"
	"testing"
)

func TestNullableThreeStates(t *testing.T) {
	for _, tc := range []struct {
		body  string
		set   bool
		value *string
	}{
		{body: `{}`},
		{body: `{"field":null}`, set: true},
		{body: `{"field":"value"}`, set: true, value: func() *string { v := "value"; return &v }()},
	} {
		var dto struct {
			Field Nullable[string] `json:"field"`
		}
		if err := json.Unmarshal([]byte(tc.body), &dto); err != nil {
			t.Fatal(err)
		}
		n := dto.Field.ToDomain()
		if n.Set != tc.set || (n.Value == nil) != (tc.value == nil) {
			t.Fatalf("%s: %+v", tc.body, n)
		}
		if n.Value != nil && *n.Value != *tc.value {
			t.Fatal("wrong value")
		}
	}
	var n Nullable[int]
	if err := json.Unmarshal([]byte(`"bad"`), &n); err == nil {
		t.Fatal("accepted wrong type")
	}
}
