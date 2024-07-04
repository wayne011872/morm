package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/wayne011872/morm"
	"github.com/wayne011872/morm/conn"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	MONGO_URI = "mongodb://mongo1:27017,mongo2:27018,mongo3:27019/?replicaSet=myReplicaSet&ssl=false&readPreference=primary"
	MONGO_DB  = "test"
	PORT      = "2112"
)

func getMongoConf() conn.MongoConf {
	return conn.MongoConf{
		Uri:       MONGO_URI,
		DefaultDB: MONGO_DB,
	}
}

func main() {
	conf := getMongoConf()
	go func() {
		for {
			ctx := context.Background()
			dbconn, err := conf.NewDefaultDbConnWithOpts(ctx)
			if err != nil {
				panic(err)
			}
			time.Sleep(time.Second * 5)
			dbconn.Close()
			time.Sleep(time.Second * 10)
		}
	}()

	prometheus.MustRegister(conn.MongoOpsQueued)
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("http://localhost:" + PORT + "/metrics")
	http.ListenAndServe(":"+PORT, nil)
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
