package classroom

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Academico struct {
	DesempenhoGeral          string `bson:"desempenho_geral" json:"desempenho_geral"`
	EvolucaoRecente          string `bson:"evolucao_recente" json:"evolucao_recente"`
	DificuldadesAprendizagem string `bson:"dificuldades_aprendizagem" json:"dificuldades_aprendizagem"`
	PontosFortesAprendizagem string `bson:"pontos_fortes_aprendizagem" json:"pontos_fortes_aprendizagem"`
}

type Emocional struct {
	EstadoEmocionalGeral string `bson:"estado_emocional_geral" json:"estado_emocional_geral"`
	Engajamento          string `bson:"engajamento" json:"engajamento"`
}

type Risco struct {
	RiscoDesempenhoBaixo string `bson:"risco_desempenho_baixo" json:"risco_desempenho_baixo"`
	RiscoDesengajamento  string `bson:"risco_desengajamento" json:"risco_desengajamento"`
	NecessitaIntervencao string `bson:"necessita_intervencao" json:"necessita_intervencao"`
}

type SaidaLLM struct {
	ResumoLLM                 string `bson:"resumo_llm" json:"resumo_llm"`
	RecomendacaoParaProfessor string `bson:"recomendacao_para_professor" json:"recomendacao_para_professor"`
	RecomendacaoParaPais      string `bson:"recomendacao_para_pais" json:"recomendacao_para_pais"`
	PlanoAcaoSugerido         string `bson:"plano_acao_sugerido" json:"plano_acao_sugerido"`
}

type StudentLLMReport struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentID            string             `bson:"student_id" json:"student_id"`
	StudentObservationID string             `bson:"student_observation_id" json:"student_observation_id"`
	PeriodoReferencia    string             `bson:"periodo_referencia" json:"periodo_referencia"`
	CreatedAt            time.Time          `bson:"created_at" json:"created_at"`
	Academico            Academico          `bson:"academico" json:"academico"`
	Emocional            Emocional          `bson:"emocional" json:"emocional"`
	Risco                Risco              `bson:"risco" json:"risco"`
	SaidaLLM             SaidaLLM           `bson:"saida_llm" json:"saida_llm"`
}
