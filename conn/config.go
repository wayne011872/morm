package conn

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type MongoDI interface {
	NewDefaultDbConn(ctx context.Context) (MongoDBConn, error)
	NewDbConn(ctx context.Context, db string) (MongoDBConn, error)
	SetAuth(user, pwd string)
	GetUri() string
	GetDb() string
}

type MongoConf struct {
	Uri       string `yaml:"uri"`
	User      string `yaml:"user"`
	Pass      string `yaml:"pass"`
	DefaultDB string `yaml:"defaul"`

	authUri string
}

func (mc *MongoConf) SetAuth(user, pwd string) {
	mc.authUri = strings.Replace(mc.Uri, "{User}", user, 1)
	mc.authUri = strings.Replace(mc.authUri, "{Pwd}", pwd, 1)
}

func (mc *MongoConf) GetDb() string {
	return mc.DefaultDB
}

func (mc *MongoConf) GetUri() string {
	if mc.authUri != "" {
		return mc.authUri
	}
	return mc.Uri
}

func (mc *MongoConf) NewDefaultDbConn(ctx context.Context) (MongoDBConn, error) {
	if mc.DefaultDB == "" {
		return nil, errors.New("mongo default db not set")
	}
	return mc.NewDbConn(ctx, mc.DefaultDB)
}

func (mc *MongoConf) NewDbConn(ctx context.Context, db string) (MongoDBConn, error) {
	uri := mc.GetUri()
	if uri == "" {
		return nil, errors.New("mongo uri not set")
	}
	if db == "" {
		return nil, errors.New("db is empty")
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetConnectTimeout(10*time.Second))
	if err != nil {
		return nil, fmt.Errorf("connect error: %w", err)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("ping fail: %w", err)
	}

	dbclt := client.Database(db)
	return &mgoClientImpl{
		ctx: ctx,
		clt: client,
		db:  dbclt,
	}, nil
}
