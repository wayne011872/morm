package conn

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	MongoOpsQueued = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "mongo_connection",
		Help: "Number of MongoDB connection.",
	})
)

type MongoOptsDI interface {
	NewDefaultDbConnWithOpts(ctx context.Context) (MongoDBConn, error)
	NewDbConnWithOpts(ctx context.Context, db string) (MongoDBConn, error)
	SetAuth(user, pwd string)
	GetUri() string
	GetDb() string
}

func (mc *MongoConf) NewDefaultDbConnWithOpts(ctx context.Context) (MongoDBConn, error) {
	return mc.NewDbConnWithOpts(ctx, mc.DefaultDB)
}

func (mc *MongoConf) NewDbConnWithOpts(ctx context.Context, db string) (MongoDBConn, error) {
	result, err := mc.NewDbConn(ctx, db)
	if err != nil {
		return nil, err
	}
	MongoOpsQueued.Inc()
	return &mgoClientOptsImpl{
		MongoDBConn: result,
	}, nil
}

type mgoClientOptsImpl struct {
	MongoDBConn
}

func (m *mgoClientOptsImpl) Close() error {
	err := m.MongoDBConn.Close()
	if err != nil {
		return err
	}
	MongoOpsQueued.Dec()
	return nil
}
