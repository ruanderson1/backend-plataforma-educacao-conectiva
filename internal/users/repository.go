package users

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const indexTimeout = 10 * time.Second

// ErrDuplicateEmailRole representa violação de unicidade por email+perfil.
var ErrDuplicateEmailRole = errors.New("duplicate email and role")

// User representa o documento de usuário persistido na coleção users.
type User struct {
	PublicID     string    `bson:"public_id"`
	Name         string    `bson:"name"`
	Email        string    `bson:"email"`
	Role         string    `bson:"role"`
	PasswordHash string    `bson:"password_hash"`
	CreatedAt    time.Time `bson:"created_at"`
}

// Repository encapsula acesso à coleção de usuários no MongoDB.
type Repository struct {
	collection *mongo.Collection
}

// NewRepository inicializa o repositório e garante índices obrigatórios.
func NewRepository(db *mongo.Database) (*Repository, error) {
	repository := &Repository{collection: db.Collection("users")}

	ctx, cancel := context.WithTimeout(context.Background(), indexTimeout)
	defer cancel()

	_, err := repository.collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}, {Key: "role", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}

	return repository, nil
}

// Create insere um novo usuário e traduz erro de duplicidade para erro de domínio.
func (r *Repository) Create(ctx context.Context, user User) error {
	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicateEmailRole
		}
		return err
	}

	return nil
}

// FindByEmailAndRole busca usuário por email e perfil.
func (r *Repository) FindByEmailAndRole(ctx context.Context, email, role string) (User, error) {
	var user User
	err := r.collection.FindOne(ctx, bson.M{"email": email, "role": role}).Decode(&user)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

// FindByPublicID busca usuário por identificador público.
func (r *Repository) FindByPublicID(ctx context.Context, publicID string) (User, error) {
	var user User
	err := r.collection.FindOne(ctx, bson.M{"public_id": publicID}).Decode(&user)
	if err != nil {
		return User{}, err
	}

	return user, nil
}
