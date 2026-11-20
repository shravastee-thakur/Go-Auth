package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

/*
bson -> name field inside atlas
json -> for user response naming
json:"-" -> same as select false
omitempty -> dont save empty filed if empty then generate
*/

type User struct {
	Id    			bson.ObjectID 	`json:"id" bson:"_id,omitempty"`
	Name  			string 			`json:"name" bson:"name"`
	Email   		string 			`json:"email" bson:"email"`
	Password 		string 			`json:"-" bson:"password"`
	RefreshToken 	string 			`json:"-" bson:"refresh_token,omitempty"`
	CreatedAt 		time.Time 		`bson:"created_at" json:"created_at"`
	UpdatedAt 		time.Time 		`bson:"updated_at" json:"updated_at"`
}