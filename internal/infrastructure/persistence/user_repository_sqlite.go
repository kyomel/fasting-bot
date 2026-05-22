package persistence

import (
	"database/sql"
	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

type UserRepositorySQLite struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	return &UserRepositorySQLite{db: db}
}

func (r *UserRepositorySQLite) Create(user *domain.User) error {
	result, err := r.db.Exec(
		"INSERT INTO users (phone, name, jid) VALUES (?, ?, ?)",
		user.Phone, user.Name, user.JID,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	user.ID = id
	return nil
}

func (r *UserRepositorySQLite) FindByPhone(phone string) (*domain.User, error) {
	var user domain.User
	err := r.db.QueryRow(
		"SELECT id, phone, name, jid, created_at FROM users WHERE phone = ?",
		phone,
	).Scan(&user.ID, &user.Phone, &user.Name, &user.JID, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositorySQLite) FindByID(id int64) (*domain.User, error) {
	var user domain.User
	err := r.db.QueryRow(
		"SELECT id, phone, name, jid, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Phone, &user.Name, &user.JID, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
