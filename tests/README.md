# Testes do Sistema

Todos os testes ficam nesta pasta (`tests/`) para facilitar execução.

## Executar testes rápidos (sem Mongo real)

```powershell
go test ./tests/...
```

## Executar também integração com Mongo

Defina a variável de ambiente e rode os mesmos testes:

```powershell
$env:MONGO_URI_TEST="mongodb+srv://..."
go test ./tests/...
```

Se `MONGO_URI_TEST` não estiver definido, os testes de integração serão marcados como `skip`.
