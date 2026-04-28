package classroom

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AgregadoAlunos struct {
	DesempenhoMedioTurma        string `bson:"desempenho_medio_turma" json:"desempenho_medio_turma"`
	PrincipaisDificuldadesTurma string `bson:"principais_dificuldades_turma" json:"principais_dificuldades_turma"`
	NivelEngajamentoTurma       string `bson:"nivel_engajamento_turma" json:"nivel_engajamento_turma"`
}

type RiscoColetivo struct {
	RiscoDesengajamentoTurma     string `bson:"risco_desengajamento_turma" json:"risco_desengajamento_turma"`
	NecessitaIntervencaoColetiva string `bson:"necessita_intervencao_coletiva" json:"necessita_intervencao_coletiva"`
}

type SaidaLLMTurma struct {
	ResumoLLMTurma                 string `bson:"resumo_llm_turma" json:"resumo_llm_turma"`
	RecomendacaoParaProfessorTurma string `bson:"recomendacao_para_professor_turma" json:"recomendacao_para_professor_turma"`
	PlanoAcaoTurma                 string `bson:"plano_acao_turma" json:"plano_acao_turma"`
}

type ClassLLMReport struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ClassID            string             `bson:"class_id" json:"class_id"`
	ClassObservationID string             `bson:"class_observation_id" json:"class_observation_id"`
	PeriodoReferencia  string             `bson:"periodo_referencia" json:"periodo_referencia"`
	CreatedAt          time.Time          `bson:"created_at" json:"created_at"`
	AgregadoAlunos     AgregadoAlunos     `bson:"agregado_alunos" json:"agregado_alunos"`
	RiscoColetivo      RiscoColetivo      `bson:"risco_coletivo" json:"risco_coletivo"`
	SaidaLLMTurma      SaidaLLMTurma      `bson:"saida_llm_turma" json:"saida_llm_turma"`
}
