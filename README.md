# Plataforma Educação Conectiva

Este repositório contém o backend da plataforma. Neste momento, a parte implementada e documentada em detalhe é o **módulo de autenticação**.

## Visão geral do projeto

- Backend em Go, consumido por frontend React
- Persistência em MongoDB
- Arquitetura em camadas (`handlers`, `service`, `repository`, `database`)

> Observação: esta documentação descreve principalmente a **seção de autenticação**, que é a parte atualmente desenvolvida.

## Estrutura de pastas

```text
cmd/
  web/
    main.go                # Bootstrap da aplicação (env, conexão, servidor)
    routes.go              # Registro de rotas e middleware HTTP
    handlers/
      auth.go              # Handlers HTTP da seção de autenticação

internal/
  auth/
    service.go             # Regras de negócio de autenticação/sessão
  users/
    repository.go          # Repositório Mongo para usuários
  database/
    mongo.go               # Conexão MongoDB (connect + ping)

tests/
  auth/                    # Testes de integração da autenticação
  handlers/                # Testes unitários dos handlers
  database/                # Teste unitário de conexão
  run-all-tests.ps1        # Script de execução rápida dos 3 testes principais
```

## Seção: Autenticação

### Objetivo

Implementar autenticação para dois perfis:
- `professor`
- `responsavel`

Com suporte a:
- cadastro
- login
- sessão por token
- endpoint de usuário logado (`/me`)
- logout

### Fluxo funcional

1. Cadastro recebe payload e valida campos por perfil.
2. Senha é protegida com `bcrypt`.
3. Usuário é persistido em `users` com índice único (`email + role`).
4. Login gera token aleatório e salva sessão em `sessions`.
5. `/api/auth/me` resolve usuário pelo token Bearer.
6. Logout remove sessão e invalida o token.

### Rotas de autenticação

- `POST /api/auth/professor/register`
- `POST /api/auth/professor/login`
- `POST /api/auth/responsavel/register`
- `POST /api/auth/responsavel/login`
- `GET /api/auth/me`
- `POST /api/auth/logout`

### Payloads esperados

Professor (cadastro):

```json
{
  "nome": "Ana",
  "email": "ana@teste.com",
  "senha": "123456"
}
```

Responsável (cadastro):

```json
{
  "email": "resp@teste.com",
  "senha": "123456"
}
```

Login (ambos):

```json
{
  "email": "ana@teste.com",
  "senha": "123456"
}
```

### Modelo de dados (Mongo)

Coleção `users`:
- `public_id`
- `name`
- `email`
- `role`
- `password_hash`
- `created_at`

Coleção `sessions`:
- `token`
- `user_id` (referência ao `public_id`)
- `created_at`

## Configuração e execução

Variáveis:
- `MONGO_URI` (obrigatória)
- `MONGO_DB` (opcional, default `plataforma_educacao_conectiva`)
- `SERVER_ADDR` (opcional, default `:4000`)

Exemplo PowerShell:

```powershell
$env:MONGO_URI="mongodb+srv://usuario:senha@cluster.mongodb.net/"
$env:MONGO_DB="plataforma_educacao_conectiva"
$env:SERVER_ADDR=":4000"
```

Executar API:

```powershell
go run ./cmd/web
```

## Testes

Execução rápida da suíte principal:

```powershell
.\tests\run-all-tests.ps1
```

Ou todos os testes da pasta `tests`:

```powershell
go test ./tests/...
```
