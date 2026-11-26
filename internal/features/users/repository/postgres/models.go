package postgres

import "github.com/makmanu/golang_todoapp/internal/core/domain"

type UserModel struct {
	ID          string
	Version     int64
	FullName    string
	PhoneNumber *string
}
type scanner interface{ Scan(...any) error }

func (m *UserModel) Scan(row scanner) error {
	return row.Scan(&m.ID, &m.Version, &m.FullName, &m.PhoneNumber)
}
func modelToDomain(m UserModel) domain.User {
	return domain.User{ID: domain.UserID(m.ID), Version: m.Version, FullName: m.FullName, PhoneNumber: m.PhoneNumber}
}
