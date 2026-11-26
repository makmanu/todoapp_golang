package domain

import (
	"errors"
	"strings"
	"testing"

	coreerrors "github.com/makmanu/golang_todoapp/internal/core/errors"
)

func ptr[T any](v T) *T { return &v }
func TestUserInvariants(t *testing.T) {
	for _, tc := range []struct {
		name     string
		fullName string
		phone    *string
		valid    bool
	}{
		{"unicode minimum", "Иван", nil, true},
		{"hundred runes", strings.Repeat("я", 100), ptr("+79991234567"), true},
		{"short", "Ян", nil, false},
		{"long", strings.Repeat("я", 101), nil, false},
		{"NUL", "ab\x00c", nil, false},
		{"phone no plus", "Name", ptr("79991234567"), false},
		{"phone letters", "Name", ptr("+7999abcdefg"), false},
		{"phone leading zero", "Name", ptr("+0123456789"), false},
		{"phone shortest", "Name", ptr("+123456789"), true},
		{"phone longest", "Name", ptr("+12345678901234"), true},
		{"phone too long", "Name", ptr("+123456789012345"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := CreateUser(tc.fullName, tc.phone)
			if err != nil {
				t.Fatal(err)
			}
			if u.Version != 1 || string(u.ID)[14] != '4' {
				t.Fatalf("invalid factory defaults: %+v", u)
			}
			err = u.Validate()
			if (err == nil) != tc.valid {
				t.Fatalf("Validate()=%v", err)
			}
			if err != nil && !errors.Is(err, coreerrors.ErrInvalidArgument) {
				t.Fatal(err)
			}
		})
	}
}
func TestApplyPatchIsAtomicAndClonesPointers(t *testing.T) {
	phone := "+79991234567"
	u, _ := CreateUser("Original", &phone)
	phone = "+79990000000"
	if *u.PhoneNumber != "+79991234567" {
		t.Fatal("factory aliases caller phone")
	}
	before := u
	err := u.ApplyPatch(UserPatch{FullName: Nullable[string]{Set: true, Value: ptr("Changed")}, PhoneNumber: Nullable[string]{Set: true, Value: ptr("bad")}})
	if !errors.Is(err, coreerrors.ErrInvalidArgument) || u != before {
		t.Fatalf("invalid patch mutated user: %+v, %v", u, err)
	}
	if err := u.ApplyPatch(UserPatch{}); err != nil || u != before {
		t.Fatal("empty patch changed fields")
	}
	if err := u.ApplyPatch(UserPatch{FullName: Nullable[string]{Set: true}}); !errors.Is(err, coreerrors.ErrInvalidArgument) {
		t.Fatal(err)
	}
	newPhone := "+79995555555"
	if err := u.ApplyPatch(UserPatch{PhoneNumber: Nullable[string]{Set: true, Value: &newPhone}}); err != nil {
		t.Fatal(err)
	}
	newPhone = "bad"
	if *u.PhoneNumber != "+79995555555" {
		t.Fatal("patch aliases caller phone")
	}
	if err := u.ApplyPatch(UserPatch{PhoneNumber: Nullable[string]{Set: true}}); err != nil || u.PhoneNumber != nil {
		t.Fatal("null did not clear phone")
	}
}
