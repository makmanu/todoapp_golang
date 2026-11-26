package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	coreerrors "github.com/makmanu/golang_todoapp/internal/core/errors"
)

type UserID string

func ParseUserID(s string) (UserID, error) {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return "", fmt.Errorf("id must be a UUID: %w", coreerrors.ErrInvalidArgument)
	}
	raw := strings.ReplaceAll(s, "-", "")
	if len(raw) != 32 {
		return "", fmt.Errorf("id must be a UUID: %w", coreerrors.ErrInvalidArgument)
	}
	if _, err := hex.DecodeString(raw); err != nil {
		return "", fmt.Errorf("id must be a UUID: %w", coreerrors.ErrInvalidArgument)
	}
	return UserID(strings.ToLower(s)), nil
}

type User struct {
	ID          UserID
	Version     int64
	FullName    string
	PhoneNumber *string
}

type UserPatch struct {
	FullName    Nullable[string]
	PhoneNumber Nullable[string]
}

var phonePattern = regexp.MustCompile(`^\+[1-9][0-9]{8,13}$`)

func CreateUser(fullName string, phoneNumber *string) (User, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return User{}, fmt.Errorf("generate user id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	id := fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
	return User{ID: UserID(id), Version: 1, FullName: fullName, PhoneNumber: cloneString(phoneNumber)}, nil
}

func (u User) Validate() error {
	if _, err := ParseUserID(string(u.ID)); err != nil {
		return fmt.Errorf("user id: %w", err)
	}
	if u.Version < 1 {
		return fmt.Errorf("version must be positive: %w", coreerrors.ErrInvalidArgument)
	}
	if !utf8.ValidString(u.FullName) || utf8.RuneCountInString(u.FullName) < 3 || utf8.RuneCountInString(u.FullName) > 100 || strings.ContainsRune(u.FullName, 0) {
		return fmt.Errorf("full_name must contain 3–100 Unicode characters without NUL: %w", coreerrors.ErrInvalidArgument)
	}
	if u.PhoneNumber != nil && !phonePattern.MatchString(*u.PhoneNumber) {
		return fmt.Errorf("phone_number must start with + and contain 9–14 digits, first digit nonzero: %w", coreerrors.ErrInvalidArgument)
	}
	return nil
}

func (p UserPatch) Validate() error {
	if p.FullName.Set && p.FullName.Value == nil {
		return fmt.Errorf("full_name cannot be null: %w", coreerrors.ErrInvalidArgument)
	}
	return nil
}

func (u *User) ApplyPatch(p UserPatch) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("validate patch: %w", err)
	}
	next := *u
	if p.FullName.Set {
		next.FullName = *p.FullName.Value
	}
	if p.PhoneNumber.Set {
		next.PhoneNumber = cloneString(p.PhoneNumber.Value)
	}
	if err := next.Validate(); err != nil {
		return fmt.Errorf("validate patched user: %w", err)
	}
	*u = next
	return nil
}

func cloneString(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}
