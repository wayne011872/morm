package morm

import (
	"github.com/wayne011872/morm/format"
	"go.mongodb.org/mongo-driver/bson"
)

func (mm *mgoModelImpl) GetPipeMatchPaginationSource(aggr MgoAggregate, q bson.M, sort bson.M) format.PaginationSource {
	return &mongoPipeMatchPaginationImpl{
		MgoDBModel: mm,
		a:          aggr,
		q:          q,
		sort:       sort,
	}
}

type mongoPipeMatchPaginationImpl struct {
	MgoDBModel
	a    MgoAggregate
	q    bson.M
	sort bson.M
}

func (mpi *mongoPipeMatchPaginationImpl) Count() (int64, error) {
	return mpi.CountAggrDocuments(mpi.a, mpi.q)
}

func (mpi *mongoPipeMatchPaginationImpl) Data(limit, p int64, f format.ObjToMapFunc) ([]map[string]interface{}, error) {
	result, err := mpi.PagePipeFind(mpi.a, mpi.q, mpi.sort, limit, p)
	if err != nil {
		return nil, err
	}
	formatResult, l := format.DocToMap(result, f)
	if l == 0 {
		return nil, nil
	}
	return formatResult.([]map[string]interface{}), nil
}
