package classroom

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StudentObservation struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentID           string             `bson:"student_id" json:"student_id"`
	PeriodoReferencia   string             `bson:"periodo_referencia" json:"periodo_referencia"`
	ObservacaoProfessor string             `bson:"observacao_professor" json:"observacao_professor"`
	ObservacaoPais      string             `bson:"observacao_pais" json:"observacao_pais"`
	CreatedAt           time.Time          `bson:"created_at" json:"created_at"`
}
