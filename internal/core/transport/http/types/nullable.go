package types

import (
	"bytes"
	"encoding/json"

	"github.com/makmanu/golang_todoapp/internal/core/domain"
)

type Nullable[T any] struct{ domain.Nullable[T] }

func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.Set = true
	n.Value = nil
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.Value = &value
	return nil
}
func (n Nullable[T]) ToDomain() domain.Nullable[T] { return n.Nullable }
