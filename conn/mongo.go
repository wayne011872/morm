package conn

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type ctxKey string

const (
	_CTX_KEY_MONGO = "ctxMongoKey"
	CtxMongoKey    = ctxKey(_CTX_KEY_MONGO)
)

type MongoDBConn interface {
	GetDbConn() *mongo.Database
	WithSession(f func(sc mongo.SessionContext) error) error
	AbortTransaction(sc mongo.SessionContext) error
	CommitTransaction(sc mongo.SessionContext) error
	Close() error
	Ping() error
}

func SetMgoDbConnToReq(req *http.Request, clt MongoDBConn) *http.Request {
	return req.WithContext(SetMgoDbConnToCtx(req.Context(), clt))
}

func GetMgoDbConnFromReq(req *http.Request) MongoDBConn {
	return GetMgoDbConnFromCtx(req.Context())
}

func SetMgoDbConnToCtx(ctx context.Context, clt MongoDBConn) context.Context {
	return context.WithValue(ctx, CtxMongoKey, clt)
}

func GetMgoDbConnFromCtx(ctx context.Context) MongoDBConn {
	cltInter := ctx.Value(CtxMongoKey)
	if dbclt, ok := cltInter.(MongoDBConn); ok {
		return dbclt
	}
	return nil
}

func SetMgoDbConnToGin(c *gin.Context, clt MongoDBConn) *gin.Context {
	c.Set(_CTX_KEY_MONGO, clt)
	return c
}

func GetMgoDbConnFromGin(c *gin.Context) MongoDBConn {
	clt, ok := c.Get(_CTX_KEY_MONGO)
	if !ok {
		return nil
	}
	return clt.(MongoDBConn)
}

type mgoClientImpl struct {
	ctx     context.Context
	clt     *mongo.Client
	db      *mongo.Database
	session mongo.Session
}

func (m *mgoClientImpl) WithSession(f func(sc mongo.SessionContext) error) error {
	if m.session != nil {
		return nil
	}
	session, err := m.clt.StartSession()
	if err != nil {
		return err
	}
	if err := session.StartTransaction(); err != nil {
		return err
	}
	m.session = session
	return mongo.WithSession(m.ctx, m.session, f)
}

func (m *mgoClientImpl) GetDBList() ([]string, error) {
	return m.clt.ListDatabaseNames(m.ctx, bson.M{})
}

func (m *mgoClientImpl) Close() error {
	if m == nil {
		return nil
	}
	if m.session != nil {
		m.session.EndSession(m.ctx)
		m.session = nil
	}
	if m.clt != nil {
		err := m.clt.Disconnect(m.ctx)
		m.clt = nil
		m.db = nil
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *mgoClientImpl) Ping() error {
	return m.clt.Ping(m.ctx, readpref.Primary())
}

func (m *mgoClientImpl) GetDbConn() *mongo.Database {
	return m.db
}

func (m *mgoClientImpl) AbortTransaction(sc mongo.SessionContext) error {
	if m.session == nil {
		return errors.New("session is nil")
	}
	err := m.session.AbortTransaction(sc)
	m.session = nil
	return err
}
func (m *mgoClientImpl) CommitTransaction(sc mongo.SessionContext) error {
	if m.session == nil {
		return errors.New("session is nil")
	}
	err := m.session.CommitTransaction(sc)
	m.session = nil
	return err
}
