# Plataforma Educação Conectiva - Backend

Backend da plataforma educacional, desenvolvido em Go com MongoDB.

## Visão geral

- API HTTP em Go
- Persistência em MongoDB
- Arquitetura em camadas (`handlers`, `service`, `repository`)
- CORS configurado para frontends locais em http://localhost:5173 e http://localhost:8081
- CORS configurável por variável CORS_ALLOWED_ORIGINS (lista separada por vírgula)

## Módulos implementados

- Autenticação (`professor` e `responsavel`)
- Perfil do professor
- Gestão de turmas (CRUD)
- Gestão de alunos (CRUD)

## Estrutura de pastas

```text
cmd/
  web/
    main.go                # Bootstrap da aplicação (env, conexão, servidor)
    routes.go              # Registro de rotas e middleware HTTP
    handlers/
      auth.go              # Handlers HTTP de autenticação
      classroom.go         # Adaptação de handlers de turma
      profile.go           # Handlers HTTP de perfil

internal/
  auth/
    service.go             # Regras de autenticação e sessão
  classroom/
    handler.go             # Handlers de turma
    service.go             # Regras de negócio de turma
    repository.go          # Persistência de turmas
    student.go             # Suporte a dados de aluno
  users/
    repository.go          # Repositório de usuários
    profile_repository.go  # Repositório de perfil
    profile_service.go     # Serviço de perfil
  database/
    mongo.go               # Conexão MongoDB (connect + ping)

tests/
  auth/                    # Testes de integração de autenticação
  handlers/                # Testes unitários de handlers
  database/                # Testes de conexão
  run-all-tests.ps1        # Script de execução rápida
```

## Rotas da API

Públicas:

- `GET /api/health`
- `POST /api/auth/professor/register`
- `POST /api/auth/professor/login`
- `POST /api/auth/responsavel/register`
- `POST /api/auth/responsavel/login`
- `GET /api/auth/me`
- `POST /api/auth/logout`

Protegidas (Bearer token):

- `GET /api/professor/profile`
- `POST /api/professor/profile`
- `GET /api/classes`
- `POST /api/classes`
- `GET /api/classes/:id`
- `PUT /api/classes/:id`
- `DELETE /api/classes/:id`
- `GET /api/students?sala=:classId`
- `POST /api/students`
- `GET /api/students/:id`
- `PUT /api/students/:id`
- `DELETE /api/students/:id`

## Exemplo de payloads

Cadastro de professor:

```json
{
  "nome": "Ana",
  "email": "ana@teste.com",
  "senha": "123456"
}
```

Cadastro de responsavel:

```json
{
  "email": "resp@teste.com",
  "senha": "123456"
}
```

Login:

```json
{
  "email": "ana@teste.com",
  "senha": "123456"
}

Cadastro de aluno:

```json
{
  "nome": "Carlos Souza",
  "notas": {
    "tipo": "Prova",
    "pontuacao": 8.5,
    "observacoes": "Bom desempenho"
  },
  "sala": "6802f5f5e2a6c5d8fb727f31",
  "role": "aluno"
}
```
```

## Configuração e execução

Requisitos:

- Go 1.22+
- MongoDB (local ou Atlas)

Variáveis de ambiente:

- `MONGO_URI` (obrigatória)
- `MONGO_DB` (opcional, default `plataforma_educacao_conectiva`)
- `SERVER_ADDR` (opcional, default `:4000`)
- `CORS_ALLOWED_ORIGINS` (opcional, ex.: `http://localhost:8081,http://localhost:5173`)

Exemplo no PowerShell:

```powershell
$env:MONGO_URI="mongodb+srv://usuario:senha@cluster.mongodb.net/"
$env:MONGO_DB="plataforma_educacao_conectiva"
$env:SERVER_ADDR=":4000"
```

Executar a API:

```powershell
go run ./cmd/web
```

Inserir 3 alunos de teste em uma turma existente:

```powershell
$env:MONGO_URI="mongodb+srv://usuario:senha@cluster.mongodb.net/"
$env:MONGO_DB="plataforma_educacao_conectiva"
# Opcional: informe a turma alvo. Se omitir, o seed usa a primeira turma encontrada.
$env:SEED_CLASSROOM_ID="6802f5f5e2a6c5d8fb727f31"
go run ./cmd/seed-students
```

## Testes

Executar testes principais:

```powershell
.\tests\run-all-tests.ps1
```

Executar toda a pasta `tests`:

```powershell
go test ./tests/...
```

Para testes de integração com Mongo real, defina `MONGO_URI_TEST`.
