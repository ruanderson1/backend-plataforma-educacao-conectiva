package users

import "time"

// Profile representa dados de perfil do professor
// (foto: URL/base64, descricao, contato)
type Profile struct {
	PublicID     string    `bson:"public_id" json:"public_id"`
	Photo        string    `bson:"photo" json:"photo"`
	Formacoes    string    `bson:"formacoes" json:"formacoes"`
	AreasAtuacao string    `bson:"areas_atuacao" json:"areas_atuacao"`
	Pesquisas    string    `bson:"pesquisas" json:"pesquisas"`
	Descricao    string    `bson:"descricao" json:"descricao"`
	Telefone     string    `bson:"telefone" json:"telefone"`
	UpdatedAt    time.Time `bson:"updated_at" json:"updated_at"`
}
