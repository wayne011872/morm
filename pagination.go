package morm

import (
	"github.com/wayne011872/morm/format"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (mm *mgoModelImpl) GetPaginationSource(d DocInter, q bson.M, opts ...*options.FindOptions) format.PaginationSource {
	return &mongoPaginationImpl{
		MgoDBModel: mm,
		d:          d,
		q:          q,
		findOpts:   opts,
	}
}

type mongoPaginationImpl struct {
	MgoDBModel
	d        DocInter
	q        bson.M
	findOpts []*options.FindOptions
}

func (mpi *mongoPaginationImpl) Count() (int64, error) {
	return mpi.CountDocuments(mpi.d, mpi.q)
}

func (mpi *mongoPaginationImpl) Data(limit, p int64, f format.ObjToMapFunc) ([]map[string]interface{}, error) {
	result, err := mpi.PageFind(mpi.d, mpi.q, limit, p, mpi.findOpts...)
	if err != nil {
		return nil, err
	}
	formatResult, l := format.DocToMap(result, f)
	if l == 0 {
		return nil, nil
	}
	return formatResult.([]map[string]interface{}), nil
}
