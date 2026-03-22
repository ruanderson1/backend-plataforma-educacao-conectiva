package classroom

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ClassRoom struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name       string             `bson:"name" json:"name"`
	AccessCode string             `bson:"accessCode" json:"accessCode"`
	YearGrade  string             `bson:"yearGrade" json:"yearGrade"`
	TeacherID  string             `bson:"teacherId" json:"teacherId"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time          `bson:"updatedAt" json:"updatedAt"`
}
