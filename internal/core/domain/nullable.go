package domain

// Nullable represents an omitted field, an explicit null, or a value.
type Nullable[T any] struct {
	Value *T
	Set   bool
}
