package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"plataforma/internal/users"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleResponsavel = "responsavel"
	RoleProfessor   = "professor"

	indexTimeout     = 10 * time.Second
	operationTimeout = 5 * time.Second
	bcryptCost       = bcrypt.DefaultCost
)

var (
	// ErrInvalidInput indica payload inválido para cadastro/login.
	ErrInvalidInput = errors.New("invalid input")
	// ErrEmailAlreadyExists indica tentativa de cadastro duplicado por email+perfil.
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrInvalidCredentials indica falha de autenticação.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUnauthorized indica token ausente, inválido ou revogado.
	ErrUnauthorized = errors.New("unauthorized")
)

// User representa os dados públicos retornados pela camada de autenticação.
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// sessionDocument representa a sessão persistida no MongoDB.
type sessionDocument struct {
	Token     string    `bson:"token"`
	UserID    string    `bson:"user_id"`
	CreatedAt time.Time `bson:"created_at"`
}

// RegisterInput descreve o payload aceito no fluxo de cadastro.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
	Role     string
}

// LoginInput descreve o payload aceito no fluxo de login.
type LoginInput struct {
	Email    string
	Password string
	Role     string
}

// Service implementa casos de uso de autenticação e sessão.
type Service struct {
	usersRepository    *users.Repository
	sessionsCollection *mongo.Collection
}

// NewService inicializa o serviço e os índices obrigatórios de persistência.
func NewService(db *mongo.Database) (*Service, error) {
	usersRepository, err := users.NewRepository(db)
	if err != nil {
		return nil, err
	}

	service := &Service{
		usersRepository:    usersRepository,
		sessionsCollection: db.Collection("sessions"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), indexTimeout)
	defer cancel()

	if _, err := service.sessionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return nil, err
	}

	return service, nil
}

// Register valida dados, cria hash de senha e persiste um novo usuário.
func (s *Service) Register(input RegisterInput) (User, error) {
	name := strings.TrimSpace(input.Name)
	email := normalizeEmail(input.Email)
	password := strings.TrimSpace(input.Password)
	role := strings.TrimSpace(input.Role)

	if email == "" || password == "" || !isValidRole(role) {
		return User{}, ErrInvalidInput
	}

	if role == RoleProfessor && name == "" {
		return User{}, ErrInvalidInput
	}

	if role == RoleResponsavel && name == "" {
		name = "Responsável"
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return User{}, err
	}

	user := users.User{
		PublicID:     generateID(),
		Name:         name,
		Email:        email,
		Role:         role,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	if err := s.usersRepository.Create(ctx, user); err != nil {
		if errors.Is(err, users.ErrDuplicateEmailRole) {
			return User{}, ErrEmailAlreadyExists
		}
		return User{}, err
	}

	return toPublicUser(user), nil
}

// Login valida credenciais e cria uma nova sessão autenticada.
func (s *Service) Login(input LoginInput) (string, User, error) {
	email := normalizeEmail(input.Email)
	password := strings.TrimSpace(input.Password)
	role := strings.TrimSpace(input.Role)

	if email == "" || password == "" || !isValidRole(role) {
		return "", User{}, ErrInvalidInput
	}

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	user, err := s.usersRepository.FindByEmailAndRole(ctx, email, role)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", User{}, ErrInvalidCredentials
		}
		return "", User{}, err
	}

	if !checkPassword(user.PasswordHash, password) {
		return "", User{}, ErrInvalidCredentials
	}

	token := generateToken()
	_, err = s.sessionsCollection.InsertOne(ctx, sessionDocument{
		Token:     token,
		UserID:    user.PublicID,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return "", User{}, err
	}

	return token, toPublicUser(user), nil
}

// Logout invalida a sessão vinculada ao token informado.
func (s *Service) Logout(token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()
	_, _ = s.sessionsCollection.DeleteOne(ctx, bson.M{"token": token})
}

// UserByToken resolve o usuário autenticado a partir do token de sessão.
func (s *Service) UserByToken(token string) (User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return User{}, ErrUnauthorized
	}

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	var session sessionDocument
	err := s.sessionsCollection.FindOne(ctx, bson.M{"token": token}).Decode(&session)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return User{}, ErrUnauthorized
		}
		return User{}, err
	}

	user, err := s.usersRepository.FindByPublicID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return User{}, ErrUnauthorized
		}
		return User{}, err
	}

	return toPublicUser(user), nil
}

// toPublicUser converte o modelo interno para DTO público da API.
func toPublicUser(user users.User) User {
	return User{
		ID:        user.PublicID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

// isValidRole define os perfis válidos aceitos pela autenticação.
func isValidRole(role string) bool {
	return role == RoleResponsavel || role == RoleProfessor
}

// normalizeEmail padroniza o email para busca e persistência consistentes.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// hashPassword gera hash seguro da senha com bcrypt.
func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

// checkPassword valida senha com bcrypt e fallback legado para hashes antigos.
func checkPassword(storedHash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err == nil {
		return true
	}

	legacyCandidate := legacyHashPassword(password)
	return legacyCandidate == storedHash
}

// legacyHashPassword mantém compatibilidade com hashes históricos em SHA-256.
func legacyHashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// generateID cria o identificador público do usuário.
func generateID() string {
	return fmt.Sprintf("usr_%d", time.Now().UnixNano())
}

// generateToken cria token aleatório para sessão.
func generateToken() string {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return fmt.Sprintf("tk_%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}
