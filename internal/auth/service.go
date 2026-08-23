package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"plataforma/internal/users"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	// ErrClassroomNotFound indica código de sala inexistente para o cadastro do responsável.
	ErrClassroomNotFound = errors.New("classroom not found")
	// ErrInvalidChildrenCodes indica códigos de filhos inválidos para a sala informada.
	ErrInvalidChildrenCodes = errors.New("invalid children codes")
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
	Name          string
	Email         string
	Password      string
	Role          string
	ClassroomCode string
	ChildrenCodes []string
	ChildrenLinks []ResponsavelChildLinkInput
}

// ResponsavelChildLinkInput representa um vinculo filho+sala no cadastro.
type ResponsavelChildLinkInput struct {
	ClassroomCode string
	ChildCode     string
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
	classrooms         *mongo.Collection
	students           *mongo.Collection
	responsaveis       *mongo.Collection
	studentLLMReports  *mongo.Collection
}

type classroomDocument struct {
	ID         primitive.ObjectID `bson:"_id"`
	Name       string             `bson:"name"`
	YearGrade  string             `bson:"yearGrade"`
	AccessCode string             `bson:"accessCode"`
}

type studentDocument struct {
	ID   primitive.ObjectID `bson:"_id"`
	Nome string             `bson:"nome"`
}

type responsavelClassroom struct {
	ID         string `bson:"id"`
	CodigoSala string `bson:"codigo_sala"`
	NomeSala   string `bson:"nome_sala"`
	AnoTurma   string `bson:"ano_turma"`
}

type responsavelChild struct {
	ID          string               `bson:"id"`
	CodigoFilho string               `bson:"codigo_filho"`
	Nome        string               `bson:"nome"`
	Sala        responsavelClassroom `bson:"sala,omitempty"`
}

// ResponsavelChildInfo representa os dados de um filho vinculados ao responsavel autenticado.
type ResponsavelChildInfo struct {
	ID           string `json:"id"`
	Nome         string `json:"nome"`
	CodigoFilho  string `json:"codigo_filho"`
	SalaNome     string `json:"sala_nome"`
	SalaAnoTurma string `json:"sala_ano_turma"`
}

type responsavelDocument struct {
	UserID    string               `bson:"user_id"`
	Nome      string               `bson:"nome"`
	Email     string               `bson:"email"`
	SenhaHash string               `bson:"senha_hash"`
	Sala      responsavelClassroom `bson:"sala"`
	Filhos    []responsavelChild   `bson:"filhos"`
	CreatedAt time.Time            `bson:"created_at"`
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
		classrooms:         db.Collection("classrooms"),
		students:           db.Collection("students"),
		responsaveis:       db.Collection("responsaveis"),
		studentLLMReports:  db.Collection("student_llm_reports"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), indexTimeout)
	defer cancel()

	if _, err := service.sessionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return nil, err
	}

	if _, err := service.responsaveis.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}},
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

	var classroomInfo responsavelClassroom
	var childrenInfo []responsavelChild
	if role == RoleResponsavel {
		if len(input.ChildrenLinks) > 0 {
			resolvedChildren, err := s.resolveResponsavelChildrenLinks(input.ChildrenLinks)
			if err != nil {
				return User{}, err
			}
			childrenInfo = resolvedChildren
			if len(childrenInfo) > 0 {
				classroomInfo = childrenInfo[0].Sala
			}
		} else {
			resolvedClassroom, resolvedChildren, err := s.resolveResponsavelLinks(input.ClassroomCode, input.ChildrenCodes)
			if err != nil {
				return User{}, err
			}
			classroomInfo = resolvedClassroom
			childrenInfo = resolvedChildren
		}
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

	if role == RoleResponsavel {
		_, err := s.responsaveis.InsertOne(ctx, responsavelDocument{
			UserID:    user.PublicID,
			Nome:      name,
			Email:     email,
			SenhaHash: passwordHash,
			Sala:      classroomInfo,
			Filhos:    childrenInfo,
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return User{}, err
		}
	}

	return toPublicUser(user), nil
}

func (s *Service) resolveResponsavelLinks(classroomCode string, childrenCodes []string) (responsavelClassroom, []responsavelChild, error) {
	classroomCode = strings.ToUpper(strings.TrimSpace(classroomCode))
	if classroomCode == "" {
		return responsavelClassroom{}, nil, ErrInvalidInput
	}

	requestedChildrenCodes := make([]string, 0, len(childrenCodes))
	for _, code := range childrenCodes {
		normalizedCode := strings.ToUpper(strings.TrimSpace(code))
		if normalizedCode != "" {
			requestedChildrenCodes = append(requestedChildrenCodes, normalizedCode)
		}
	}
	if len(requestedChildrenCodes) == 0 {
		return responsavelClassroom{}, nil, ErrInvalidChildrenCodes
	}

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	var classroom classroomDocument
	err := s.classrooms.FindOne(ctx, bson.M{"accessCode": classroomCode}).Decode(&classroom)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return responsavelClassroom{}, nil, ErrClassroomNotFound
		}
		return responsavelClassroom{}, nil, err
	}

	studentsFilter := bson.M{
		"$or": []bson.M{
			{"sala": classroom.ID},
			{"sala": classroom.ID.Hex()},
		},
	}

	cursor, err := s.students.Find(ctx, studentsFilter)
	if err != nil {
		return responsavelClassroom{}, nil, err
	}
	defer cursor.Close(ctx)

	studentByCode := map[string]studentDocument{}
	for cursor.Next(ctx) {
		var student studentDocument
		if err := cursor.Decode(&student); err != nil {
			return responsavelClassroom{}, nil, err
		}
		studentCode := childCodeFromID(student.ID)
		studentByCode[studentCode] = student
	}
	if err := cursor.Err(); err != nil {
		return responsavelClassroom{}, nil, err
	}

	childrenInfo := make([]responsavelChild, 0, len(requestedChildrenCodes))
	for _, childCode := range requestedChildrenCodes {
		student, exists := studentByCode[childCode]
		if !exists {
			return responsavelClassroom{}, nil, ErrInvalidChildrenCodes
		}
		childrenInfo = append(childrenInfo, responsavelChild{
			ID:          student.ID.Hex(),
			CodigoFilho: childCode,
			Nome:        student.Nome,
		})
	}

	sort.Slice(childrenInfo, func(i, j int) bool {
		return childrenInfo[i].CodigoFilho < childrenInfo[j].CodigoFilho
	})

	return responsavelClassroom{
		ID:         classroom.ID.Hex(),
		CodigoSala: classroom.AccessCode,
		NomeSala:   classroom.Name,
		AnoTurma:   classroom.YearGrade,
	}, childrenInfo, nil
}

func (s *Service) resolveResponsavelChildrenLinks(childrenLinks []ResponsavelChildLinkInput) ([]responsavelChild, error) {
	if len(childrenLinks) == 0 {
		return nil, ErrInvalidChildrenCodes
	}

	classroomChildrenCodes := map[string][]string{}
	for _, link := range childrenLinks {
		classroomCode := strings.ToUpper(strings.TrimSpace(link.ClassroomCode))
		childCode := strings.ToUpper(strings.TrimSpace(link.ChildCode))
		if classroomCode == "" || childCode == "" {
			return nil, ErrInvalidInput
		}

		codes := classroomChildrenCodes[classroomCode]
		duplicate := false
		for _, existingCode := range codes {
			if existingCode == childCode {
				duplicate = true
				break
			}
		}
		if !duplicate {
			classroomChildrenCodes[classroomCode] = append(classroomChildrenCodes[classroomCode], childCode)
		}
	}

	classroomCodes := make([]string, 0, len(classroomChildrenCodes))
	for classroomCode := range classroomChildrenCodes {
		classroomCodes = append(classroomCodes, classroomCode)
	}
	sort.Strings(classroomCodes)

	childrenInfo := make([]responsavelChild, 0)
	for _, classroomCode := range classroomCodes {
		classroomInfo, childrenByClassroom, err := s.resolveResponsavelLinks(classroomCode, classroomChildrenCodes[classroomCode])
		if err != nil {
			return nil, err
		}
		for _, child := range childrenByClassroom {
			child.Sala = classroomInfo
			childrenInfo = append(childrenInfo, child)
		}
	}

	sort.Slice(childrenInfo, func(i, j int) bool {
		if childrenInfo[i].Sala.CodigoSala == childrenInfo[j].Sala.CodigoSala {
			return childrenInfo[i].CodigoFilho < childrenInfo[j].CodigoFilho
		}
		return childrenInfo[i].Sala.CodigoSala < childrenInfo[j].Sala.CodigoSala
	})

	return childrenInfo, nil
}

func childCodeFromID(id primitive.ObjectID) string {
	hexID := strings.ToUpper(id.Hex())
	if len(hexID) <= 4 {
		return hexID
	}
	return hexID[len(hexID)-4:]
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

// GetResponsavelChildren retorna apenas os filhos vinculados ao responsavel autenticado.
func (s *Service) GetResponsavelChildren(userID string) ([]ResponsavelChildInfo, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrUnauthorized
	}

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	var doc responsavelDocument
	err := s.responsaveis.FindOne(ctx, bson.M{"user_id": userID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	children := make([]ResponsavelChildInfo, 0, len(doc.Filhos))
	for _, child := range doc.Filhos {
		salaInfo := child.Sala
		if strings.TrimSpace(salaInfo.CodigoSala) == "" {
			salaInfo = doc.Sala
		}

		children = append(children, ResponsavelChildInfo{
			ID:           child.ID,
			Nome:         child.Nome,
			CodigoFilho:  child.CodigoFilho,
			SalaNome:     salaInfo.NomeSala,
			SalaAnoTurma: salaInfo.AnoTurma,
		})
	}

	sort.Slice(children, func(i, j int) bool {
		return strings.ToLower(children[i].Nome) < strings.ToLower(children[j].Nome)
	})

	return children, nil
}

// AddResponsavelChildren adiciona novos vinculos filho+sala para o responsavel autenticado.
func (s *Service) AddResponsavelChildren(userID string, links []ResponsavelChildLinkInput) ([]ResponsavelChildInfo, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrUnauthorized
	}
	if len(links) == 0 {
		return nil, ErrInvalidInput
	}

	newChildren, err := s.resolveResponsavelChildrenLinks(links)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	var doc responsavelDocument
	err = s.responsaveis.FindOne(ctx, bson.M{"user_id": userID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	mergedChildren := make([]responsavelChild, 0, len(doc.Filhos)+len(newChildren))
	existingByID := map[string]struct{}{}

	for _, child := range doc.Filhos {
		if strings.TrimSpace(child.Sala.CodigoSala) == "" {
			child.Sala = doc.Sala
		}
		mergedChildren = append(mergedChildren, child)
		existingByID[strings.ToLower(strings.TrimSpace(child.ID))] = struct{}{}
	}

	for _, child := range newChildren {
		if _, exists := existingByID[strings.ToLower(strings.TrimSpace(child.ID))]; exists {
			continue
		}
		mergedChildren = append(mergedChildren, child)
		existingByID[strings.ToLower(strings.TrimSpace(child.ID))] = struct{}{}
	}

	baseSala := doc.Sala
	if strings.TrimSpace(baseSala.CodigoSala) == "" && len(mergedChildren) > 0 {
		baseSala = mergedChildren[0].Sala
	}

	_, err = s.responsaveis.UpdateOne(ctx, bson.M{"user_id": userID}, bson.M{"$set": bson.M{
		"sala":   baseSala,
		"filhos": mergedChildren,
	}})
	if err != nil {
		return nil, err
	}

	return s.GetResponsavelChildren(userID)
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
