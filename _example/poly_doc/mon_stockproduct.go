package main

import (
	"github.com/wayne011872/morm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	stockproductC = "stockproduct"
)

type StockProduct struct {
	ID               primitive.ObjectID `bson:"_id"`
	PolyStockProduct `bson:",inline"`
}

func (f *StockProduct) UnmarshalBSON(b []byte) error {
	idStruct := struct {
		ID primitive.ObjectID `bson:"_id"`
	}{}
	err := bson.Unmarshal(b, &idStruct)
	if err != nil {
		return err
	}
	f.ID = idStruct.ID
	return f.PolyStockProduct.unmarshal(b, bson.Unmarshal)
}

func (s *StockProduct) GetType() ProductType {
	return s.PolyStockProduct.GetType()
}

func (s *StockProduct) GetShape() ShapeType {
	return s.PolyStockProduct.GetShape()
}

func (s *StockProduct) GetName() string {
	return s.PolyStockProduct.GetName()
}

func (s *StockProduct) GetState() StockProductState {
	return s.PolyStockProduct.GetState()
}

func (s *StockProduct) Validate() error {
	return s.PolyStockProduct.Validate()
}

func (s *StockProduct) GetUpdateFields() bson.D {
	return s.PolyStockProduct.GetUpdateFields()
}

func (s *StockProduct) GetID() interface{} {
	return s.ID
}

func (s *StockProduct) GetC() string {
	return stockproductC
}

func (s *StockProduct) GetDoc() interface{} {
	return s
}

func (s *StockProduct) GetIndexes() []mongo.IndexModel {
	return []mongo.IndexModel{}
}

func (s *StockProduct) AddRecord(u morm.LogUser, msg string) []*morm.Record {
	return nil
}

func (s *StockProduct) SetCreator(u morm.LogUser) {}
