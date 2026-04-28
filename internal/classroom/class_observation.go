package classroom

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ClassObservation struct {
	ID                       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClassID                  string             `bson:"class_id" json:"class_id"`
	PeriodoReferencia        string             `bson:"periodo_referencia" json:"periodo_referencia"`
	ObservacaoProfessorTurma string             `bson:"observacao_professor_turma" json:"observacao_professor_turma"`
	CreatedAt                time.Time          `bson:"created_at" json:"created_at"`
}
