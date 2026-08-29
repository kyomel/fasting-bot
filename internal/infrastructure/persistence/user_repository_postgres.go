package persistence

import (
	"database/sql"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

type UserRepositoryPostgres struct {
	db              *sql.DB
	findByPhoneStmt *sql.Stmt
	findByIDStmt    *sql.Stmt
	createStmt      *sql.Stmt
	updateNameStmt  *sql.Stmt
}

func NewUserRepositoryPostgres(db *sql.DB) repository.UserRepository {
	r := &UserRepositoryPostgres{db: db}

	r.findByPhoneStmt, _ = db.Prepare("SELECT id, username, password_hash, phone, email, name, jid, created_at FROM users WHERE phone = $1")
	r.findByIDStmt, _ = db.Prepare("SELECT id, username, password_hash, phone, email, name, jid, created_at FROM users WHERE id = $1")
	r.createStmt, _ = db.Prepare("INSERT INTO users (username, password_hash, phone, email, name, jid) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id")
	r.updateNameStmt, _ = db.Prepare("UPDATE users SET name = $1 WHERE id = $2")

	return r
}

func (r *UserRepositoryPostgres) Create(user *domain.User) error {
	var username, passwordHash, email sql.NullString
	if user.Username != "" {
		username = sql.NullString{String: user.Username, Valid: true}
	}
	if user.PasswordHash != "" {
		passwordHash = sql.NullString{String: user.PasswordHash, Valid: true}
	}
	if user.Email != "" {
		email = sql.NullString{String: user.Email, Valid: true}
	}

	err := r.createStmt.QueryRow(username, passwordHash, user.Phone, email, user.Name, user.JID).Scan(&user.ID)
	if err != nil {
		return err
	}
	return nil
}

func (r *UserRepositoryPostgres) UpdateName(userID domain.ID, name string) error {
	_, err := r.updateNameStmt.Exec(name, string(userID))
	return err
}

func scanUser(row *sql.Row) (*domain.User, error) {
	var user domain.User
	var username, passwordHash, email, name, jid sql.NullString
	if err := row.Scan(&user.ID, &username, &passwordHash, &user.Phone, &email, &name, &jid, &user.CreatedAt); err != nil {
		return nil, err
	}
	user.Username = username.String
	user.PasswordHash = passwordHash.String
	user.Email = email.String
	user.Name = name.String
	user.JID = jid.String
	return &user, nil
}

func (r *UserRepositoryPostgres) FindByPhone(phone string) (*domain.User, error) {
	return scanUser(r.findByPhoneStmt.QueryRow(phone))
}

func (r *UserRepositoryPostgres) FindByID(id domain.ID) (*domain.User, error) {
	return scanUser(r.findByIDStmt.QueryRow(string(id)))
}
