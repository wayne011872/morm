package morm

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/wayne011872/morm/conn"
	"github.com/wayne011872/morm/format"
)

type MgoAggregate interface {
	GetPipeline(q bson.M) mongo.Pipeline
	Collection
}

type MgoDBModel interface {
	DisableCheckBeforeSave(b bool)
	SetDB(db *mongo.Database)
	BatchUpdate(doclist []DocInter, getField func(d DocInter) bson.D, u LogUser) (failed []DocInter, err error)
	BatchSave(doclist []DocInter, u LogUser) (inserted []interface{}, failed []DocInter, err error)
	Save(d DocInter, u LogUser) (interface{}, error)
	RemoveAll(d DocInter, q primitive.M, u LogUser) (int64, error)
	RemoveByID(d DocInter, u LogUser) (int64, error)
	UpdateOne(d DocInter, fields bson.D, u LogUser) (int64, error)
	UpdateAll(d DocInter, q bson.M, fields bson.D, u LogUser) (int64, error)
	UnsetFields(d DocInter, q bson.M, fields []string, u LogUser) (int64, error)
	Upsert(d DocInter, u LogUser) (interface{}, error)
	FindByID(d DocInter) error
	FindOne(d DocInter, q bson.M, option ...*options.FindOneOptions) error
	Find(d DocInter, q bson.M, option ...*options.FindOptions) (interface{}, error)
	FindAndExec(
		d DocInter, q bson.M,
		exec func(i interface{}) error,
		opts ...*options.FindOptions,
	) error
	PipeFindOne(aggr MgoAggregate, filter bson.M) error
	PipeFind(aggr MgoAggregate, filter bson.M, opts ...*options.AggregateOptions) (interface{}, error)
	PipeFindAndExec(
		aggr MgoAggregate, q bson.M,
		exec func(i interface{}) error,
		opts ...*options.AggregateOptions,
	) error
	PagePipeFind(aggr MgoAggregate, filter bson.M, sort bson.M, limit, page int64) (interface{}, error)
	PageFind(d DocInter, q bson.M, limit, page int64, opts ...*options.FindOptions) (interface{}, error)

	CountDocuments(d Collection, q bson.M) (int64, error)
	GetPaginationSource(d DocInter, q bson.M, opts ...*options.FindOptions) format.PaginationSource
	GetPipePaginationSource(aggr MgoAggregate, q bson.M, sort bson.M) format.PaginationSource

	CreateCollection(dlist ...DocInter) error
	//Reference to customer code, use for aggregate pagination
	CountAggrDocuments(aggr MgoAggregate, q bson.M) (int64, error)
	GetPipeMatchPaginationSource(aggr MgoAggregate, q bson.M, sort bson.M) format.PaginationSource

	NewFindMgoDS(d DocInter, q bson.M, opts ...*options.FindOptions) MgoDS
	NewPipeFindMgoDS(d MgoAggregate, q bson.M, opts ...*options.AggregateOptions) MgoDS
}

func NewMgoModel(ctx context.Context, db *mongo.Database) MgoDBModel {
	return &mgoModelImpl{
		db:      db,
		ctx:     ctx,
		selfCtx: context.Background(),
	}
}

func NewMgoModelByReq(req *http.Request) MgoDBModel {
	mgodbclt := conn.GetMgoDbConnFromReq(req)
	if mgodbclt == nil {
		panic("database not set in req")
	}
	return &mgoModelImpl{
		db:      mgodbclt.GetDbConn(),
		ctx:     req.Context(),
		selfCtx: context.Background(),
	}
}

type mgoModelImpl struct {
	disableCheckBeforeSave bool
	db                     *mongo.Database
	ctx                    context.Context

	selfCtx context.Context
}

func (mm *mgoModelImpl) DisableCheckBeforeSave(b bool) {
	mm.disableCheckBeforeSave = b
}

func (mm *mgoModelImpl) SetDB(db *mongo.Database) {
	mm.db = db
}

func (mm *mgoModelImpl) FindAndExec(
	d DocInter, q bson.M,
	exec func(i interface{}) error,
	opts ...*options.FindOptions,
) error {
	var err error
	collection := mm.db.Collection(d.GetC())
	sortCursor, err := collection.Find(mm.ctx, q, opts...)
	if err != nil {
		return nil
	}
	val := reflect.ValueOf(d)
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	var newValue reflect.Value
	var newDoc DocInter
	for sortCursor.Next(mm.ctx) {
		newValue = reflect.New(val.Type())
		newDoc = newValue.Interface().(DocInter)
		err = sortCursor.Decode(newDoc)
		if err != nil {
			return err
		}
		err = exec(newDoc)
		if err != nil {
			return err
		}
	}
	w2 := reflect.ValueOf(newValue)
	if w2.IsZero() {
		return nil
	}
	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		f.Set(newValue.Elem().Field(i))
	}
	return err
}

func (mm *mgoModelImpl) CountDocuments(d Collection, q bson.M) (int64, error) {
	return mm.db.Collection(d.GetC()).CountDocuments(mm.ctx, q)
}

func (mm *mgoModelImpl) isCollectExisted(d DocInter) bool {
	names, err := mm.db.ListCollectionNames(mm.selfCtx, bson.D{{Key: "name", Value: d.GetC()}})
	if ce, ok := err.(mongo.CommandError); ok {
		return ce.Name == "OperationNotSupportedInTransaction"
	}
	return isStrInList(d.GetC(), names...)
}

func (mm *mgoModelImpl) CreateCollection(dlist ...DocInter) (err error) {
	for _, d := range dlist {
		// check collection exist
		if !mm.isCollectExisted(d) {
			if len(d.GetIndexes()) > 0 {
				_, err = mm.db.Collection(d.GetC()).Indexes().CreateMany(mm.ctx, d.GetIndexes())
			} else {
				err = mm.db.CreateCollection(mm.ctx, d.GetC())
			}
			if err != nil {
				return err
			}
		}
	}
	return
}

func (mm *mgoModelImpl) BatchUpdate(doclist []DocInter, getField func(d DocInter) bson.D, u LogUser) (failed []DocInter, err error) {
	if len(doclist) == 0 {
		return
	}
	collection := mm.db.Collection(doclist[0].GetC())
	var operations []mongo.WriteModel
	for _, d := range doclist {
		op := mongo.NewUpdateOneModel()

		op.SetFilter(bson.M{"_id": d.GetID()})
		op.SetUpdate(bson.D{
			{Key: "$set", Value: getField(d)},
		})
		op.SetUpsert(true)
		operations = append(operations, op)
	}
	bulkOption := options.BulkWriteOptions{}
	_, err = collection.BulkWrite(mm.ctx, operations, &bulkOption)

	if excep, ok := err.(mongo.BulkWriteException); ok {
		for _, e := range excep.WriteErrors {
			failed = append(failed, doclist[e.Index])
		}
	}
	return
}

func (mm *mgoModelImpl) BatchSave(doclist []DocInter, u LogUser) (inserted []interface{}, failed []DocInter, err error) {
	if len(doclist) == 0 {
		inserted = nil
		return
	}
	collection := mm.db.Collection(doclist[0].GetC())
	if !mm.disableCheckBeforeSave {
		err := mm.CreateCollection(doclist[0])
		if err != nil {
			return nil, doclist, err
		}
	}
	ordered := false
	var batch []interface{}
	for _, d := range doclist {
		if u != nil {
			d.SetCreator(u)
		}
		batch = append(batch, d)
	}
	var result *mongo.InsertManyResult
	result, err = collection.InsertMany(mm.ctx, batch, &options.InsertManyOptions{Ordered: &ordered})
	if result != nil {
		inserted = result.InsertedIDs
	}

	if excep, ok := err.(mongo.BulkWriteException); ok {
		for _, e := range excep.WriteErrors {
			failed = append(failed, doclist[e.Index])
		}
	}
	return
}

func (mm *mgoModelImpl) Save(d DocInter, u LogUser) (interface{}, error) {
	if !mm.disableCheckBeforeSave {
		err := mm.CreateCollection(d)
		if err != nil {
			return primitive.NilObjectID, err
		}
	}

	if u != nil {
		d.SetCreator(u)
	}
	collection := mm.db.Collection(d.GetC())

	result, err := collection.InsertOne(mm.ctx, d.GetDoc())
	if err != nil {
		return primitive.NilObjectID, err
	}
	return result.InsertedID, err

}

func (mm *mgoModelImpl) RemoveAll(d DocInter, q primitive.M, u LogUser) (int64, error) {
	collection := mm.db.Collection(d.GetC())
	result, err := collection.DeleteMany(mm.ctx, q)
	return result.DeletedCount, err
}

func (mm *mgoModelImpl) RemoveByID(d DocInter, u LogUser) (int64, error) {
	collection := mm.db.Collection(d.GetC())
	result, err := collection.DeleteOne(mm.ctx, bson.M{"_id": d.GetID()})
	return result.DeletedCount, err
}

func (mm *mgoModelImpl) UpdateOne(d DocInter, fields bson.D, u LogUser) (int64, error) {
	if u != nil {
		fields = append(fields, primitive.E{Key: "records", Value: d.AddRecord(u, "updated")})
	}
	collection := mm.db.Collection(d.GetC())
	result, err := collection.UpdateOne(mm.ctx, bson.M{"_id": d.GetID()},
		bson.D{
			{Key: "$set", Value: fields},
		},
	)
	if result != nil {
		return result.ModifiedCount, err
	}
	return 0, err
}

func (mm *mgoModelImpl) UpdateAll(d DocInter, q bson.M, fields bson.D, u LogUser) (int64, error) {
	updated := bson.D{
		{Key: "$set", Value: fields},
	}
	if u != nil {
		updated = append(updated, primitive.E{Key: "$push", Value: primitive.M{"records": NewRecord(time.Now(), u.GetAccount(), u.GetName(), "updated")}})
	}
	collection := mm.db.Collection(d.GetC())
	result, err := collection.UpdateMany(mm.ctx, q, updated)
	if result != nil {
		return result.ModifiedCount, err
	}
	return 0, err
}

func (mm *mgoModelImpl) UnsetFields(d DocInter, q bson.M, fields []string, u LogUser) (int64, error) {
	collection := mm.db.Collection(d.GetC())
	m := primitive.M{}
	for _, k := range fields {
		m[k] = ""
	}
	result, err := collection.UpdateMany(mm.ctx, q,
		bson.D{
			{Key: "$unset", Value: m},
		},
	)
	if result != nil {
		return result.ModifiedCount, err
	}
	return 0, err
}

func (mm *mgoModelImpl) Upsert(d DocInter, u LogUser) (interface{}, error) {
	err := mm.CreateCollection(d)
	if err != nil {
		return primitive.NilObjectID, err
	}

	collection := mm.db.Collection(d.GetC())
	_, err = collection.UpdateOne(mm.ctx, bson.M{"_id": d.GetID()}, bson.M{"$set": d.GetDoc()}, options.Update().SetUpsert(true))

	if err != nil {
		return primitive.NilObjectID, err
	}
	return d.GetID(), nil
}

func (mm *mgoModelImpl) FindByID(d DocInter) error {
	return mm.FindOne(d, bson.M{"_id": d.GetID()})
}

func (mm *mgoModelImpl) FindOne(d DocInter, q bson.M, option ...*options.FindOneOptions) error {
	if mm.db == nil {
		return errors.New("db is nil")
	}
	if d == nil {
		return errors.New("doc is nil")
	}
	collection := mm.db.Collection(d.GetC())
	return collection.FindOne(mm.ctx, q, option...).Decode(d)
}

func (mm *mgoModelImpl) Find(d DocInter, q bson.M, option ...*options.FindOptions) (interface{}, error) {
	myType := reflect.TypeOf(d)
	slice := reflect.MakeSlice(reflect.SliceOf(myType), 0, 0).Interface()
	collection := mm.db.Collection(d.GetC())
	sortCursor, err := collection.Find(mm.ctx, q, option...)
	if err != nil {
		return nil, err
	}
	err = sortCursor.All(mm.ctx, &slice)
	if err != nil {
		return nil, err
	}
	return slice, err
}

func (mm *mgoModelImpl) PipeFind(aggr MgoAggregate, filter bson.M, opts ...*options.AggregateOptions) (interface{}, error) {
	myType := reflect.TypeOf(aggr)
	slice := reflect.MakeSlice(reflect.SliceOf(myType), 0, 0).Interface()
	collection := mm.db.Collection(aggr.GetC())
	sortCursor, err := collection.Aggregate(mm.ctx, aggr.GetPipeline(filter), opts...)
	if err != nil {
		return nil, err
	}
	err = sortCursor.All(mm.ctx, &slice)
	if err != nil {
		return nil, err
	}
	return slice, err
}

func (mm *mgoModelImpl) PipeFindAndExec(aggr MgoAggregate, filter bson.M, exec func(i interface{}) error, opts ...*options.AggregateOptions) error {
	collection := mm.db.Collection(aggr.GetC())
	sortCursor, err := collection.Aggregate(mm.ctx, aggr.GetPipeline(filter), opts...)
	if err != nil {
		return err
	}
	val := reflect.ValueOf(aggr)
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}
	var newValue reflect.Value
	var newDoc DocInter
	for sortCursor.Next(mm.ctx) {
		newValue = reflect.New(val.Type())
		newDoc = newValue.Interface().(DocInter)
		err = sortCursor.Decode(newDoc)
		if err != nil {
			return err
		}
		err = exec(newDoc)
		if err != nil {
			return err
		}
	}

	w2 := reflect.ValueOf(newValue)
	if w2.IsZero() {
		return nil
	}
	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		f.Set(newValue.Elem().Field(i))
	}
	return err
}

func (mm *mgoModelImpl) PipeFindOne(aggr MgoAggregate, filter bson.M) error {
	collection := mm.db.Collection(aggr.GetC())
	sortCursor, err := collection.Aggregate(mm.ctx, aggr.GetPipeline(filter))
	if err != nil {
		return err
	}
	if sortCursor.Next(mm.ctx) {
		err = sortCursor.Decode(aggr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mm *mgoModelImpl) PageFind(d DocInter, filter bson.M, limit, page int64, opts ...*options.FindOptions) (interface{}, error) {
	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	skip := limit * (page - 1)
	findopt := options.Find().SetSkip(skip).SetLimit(limit)
	opts = append(opts, findopt)
	myType := reflect.TypeOf(d)
	slice := reflect.MakeSlice(reflect.SliceOf(myType), 0, 0).Interface()
	collection := mm.db.Collection(d.GetC())
	sortCursor, err := collection.Find(mm.ctx, filter, opts...)
	if err != nil {
		return nil, err
	}

	err = sortCursor.All(mm.ctx, &slice)
	return slice, err
}

func (mm *mgoModelImpl) PagePipeFind(aggr MgoAggregate, filter bson.M, sort bson.M, limit, page int64) (interface{}, error) {
	if limit <= 0 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	skip := limit * (page - 1)
	myType := reflect.TypeOf(aggr)
	slice := reflect.MakeSlice(reflect.SliceOf(myType), 0, 0).Interface()

	collection := mm.db.Collection(aggr.GetC())
	pl := append(aggr.GetPipeline(filter), bson.D{{Key: "$sort", Value: sort}}, bson.D{{Key: "$skip", Value: skip}}, bson.D{{Key: "$limit", Value: limit}})
	sortCursor, err := collection.Aggregate(mm.ctx, pl)
	if err != nil {
		return nil, err
	}
	err = sortCursor.All(mm.ctx, &slice)
	if err != nil {
		return nil, err
	}
	return slice, err
}

// ----- New added code -----

func (mm *mgoModelImpl) AggrCountDocuments(aggr MgoAggregate, q bson.M) (int64, error) {
	return mm.db.Collection(aggr.GetC()).CountDocuments(mm.ctx, q)
}

type countMgoAggregate struct {
	Count int
}

func (mm *mgoModelImpl) CountAggrDocuments(aggr MgoAggregate, q bson.M) (int64, error) {
	collection := mm.db.Collection(aggr.GetC())
	pl := append(aggr.GetPipeline(q), bson.D{{Key: "$count", Value: "count"}})
	sortCursor, err := collection.Aggregate(mm.ctx, pl)
	if err != nil {
		return 0, err
	}
	var obj countMgoAggregate
	if sortCursor.Next(mm.ctx) {
		err = sortCursor.Decode(&obj)
		if err != nil {
			return 0, err
		}
	}
	return int64(obj.Count), nil
}

func isStrInList(input string, target ...string) bool {
	for _, paramName := range target {
		if input == paramName {
			return true
		}
	}
	return false
}
