package persistence

import (
	"database/sql"
	"errors"
	"strconv"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

type UserRepositorySQLite struct {
	db              *sql.DB
	findByPhoneStmt *sql.Stmt
	findByIDStmt    *sql.Stmt
	createStmt      *sql.Stmt
	updateNameStmt  *sql.Stmt
}

func NewUserRepository(db *sql.DB) repository.UserRepository {
	r := &UserRepositorySQLite{db: db}

	r.findByPhoneStmt, _ = db.Prepare("SELECT id, phone, name, jid, created_at FROM users WHERE phone = ?")
	r.findByIDStmt, _ = db.Prepare("SELECT id, phone, name, jid, created_at FROM users WHERE id = ?")
	r.createStmt, _ = db.Prepare("INSERT INTO users (phone, name, jid) VALUES (?, ?, ?)")
	r.updateNameStmt, _ = db.Prepare("UPDATE users SET name = ? WHERE id = ?")

	return r
}

func (r *UserRepositorySQLite) Create(user *domain.User) error {
	result, err := r.createStmt.Exec(user.Phone, user.Name, user.JID)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = domain.ID(strconv.FormatInt(id, 10))
	return nil
}

func (r *UserRepositorySQLite) UpdateName(userID domain.ID, name string) error {
	_, err := r.updateNameStmt.Exec(name, string(userID))
	return err
}

func (r *UserRepositorySQLite) FindByPhone(phone string) (*domain.User, error) {
	var user domain.User
	err := r.findByPhoneStmt.QueryRow(phone).Scan(&user.ID, &user.Phone, &user.Name, &user.JID, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositorySQLite) FindByID(id domain.ID) (*domain.User, error) {
	var user domain.User
	err := r.findByIDStmt.QueryRow(string(id)).Scan(&user.ID, &user.Phone, &user.Name, &user.JID, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}
