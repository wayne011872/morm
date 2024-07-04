package main

import (
	"context"
	"fmt"

	"github.com/wayne011872/morm"
	"github.com/wayne011872/morm/conn"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func getMongoConf() conn.MongoConf {
	return conn.MongoConf{
		Uri:       "mongodb://mongo1:27017,mongo2:27018,mongo3:27019/?replicaSet=myReplicaSet&ssl=false&readPreference=primary",
		DefaultDB: "test",
	}
}

func main() {
	conf := getMongoConf()
	ctx := context.Background()
	dbconn, err := conf.NewDefaultDbConn(ctx)
	if err != nil {
		panic(err)
	}
	defer dbconn.Close()
	mgom := morm.NewMgoModel(ctx, dbconn.GetDbConn())
	id, err := mgom.Save(&User{
		ID:   primitive.NewObjectID(),
		Name: "peter",
	}, nil)
	fmt.Println(id, err)
}

type User struct {
	ID             primitive.ObjectID `bson:"_id"`
	Name           string
	morm.CommonDoc `bson:",inline"`
}

func (d *User) GetC() string {
	return "user"
}

func (d *User) GetID() interface{} {
	return d.ID
}

func (d *User) GetDoc() interface{} {
	return d
}

func (d *User) GetIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{}
}
