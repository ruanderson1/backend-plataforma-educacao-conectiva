package classroom

type Student struct {
	ID        string `bson:"_id" json:"_id"`
	Name      string `bson:"name" json:"name"`
	Email     string `bson:"email" json:"email"`
	Classroom string `bson:"classroom" json:"classroom"`
}

// StudentRepo de exemplo (pode ser expandido conforme necessidade)
type StudentRepo struct{}

func NewStudentRepo() *StudentRepo {
	return &StudentRepo{}
}
