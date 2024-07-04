package main

import (
	"context"
	"fmt"

	"github.com/wayne011872/morm"
	"github.com/wayne011872/morm/conn"
	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	fmt.Println("aaa")
	conf := getMongoConf()
	ctx := context.Background()
	dbconn, err := conf.NewDefaultDbConn(ctx)
	if err != nil {
		panic(err)
	}
	defer dbconn.Close()
	mgom := morm.NewMgoModel(ctx, dbconn.GetDbConn())
	result, err := mgom.Find(&StockProduct{}, bson.M{})
	fmt.Println(result, err)
	products := result.([]*StockProduct)
	for _, p := range products {
		fmt.Println(p.ID)
		fmt.Println(p.ToStockProductMetrial().Name)
	}
}

func getMongoConf() conn.MongoConf {
	return conn.MongoConf{
		Uri:       "mongodb://mongo1:27017,mongo2:27018,mongo3:27019/?replicaSet=myReplicaSet&ssl=false&readPreference=primary",
		DefaultDB: "test",
	}
}
