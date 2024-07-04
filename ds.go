package morm

import (
	"encoding/csv"
	"io"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MgoDS interface {
	Exec(exec func(i interface{}) error) error
	ExportCSV(w io.Writer, title []string, exec func(writer *csv.Writer, i interface{}) error) error
}

func (mm *mgoModelImpl) NewFindMgoDS(d DocInter, q bson.M, opts ...*options.FindOptions) MgoDS {
	return &findDsImpl{
		MgoDBModel: mm,
		d:          d,
		q:          q,
		opts:       opts,
	}
}

type findDsImpl struct {
	MgoDBModel
	d    DocInter
	q    bson.M
	opts []*options.FindOptions
}

func (mm *findDsImpl) Exec(exec func(i interface{}) error) error {
	return mm.FindAndExec(mm.d, mm.q, exec, mm.opts...)
}

func (mm *findDsImpl) ExportCSV(w io.Writer, title []string, exec func(writer *csv.Writer, i interface{}) error) error {
	csvWriter := csv.NewWriter(w)
	err := csvWriter.Write(title)
	defer csvWriter.Flush()
	if err != nil {
		return err
	}
	return mm.FindAndExec(mm.d, mm.q, func(i interface{}) error {
		return exec(csvWriter, i)
	}, mm.opts...)
}

func (mm *mgoModelImpl) NewPipeFindMgoDS(d MgoAggregate, q bson.M, opts ...*options.AggregateOptions) MgoDS {
	return &pipeFindDsImpl{
		MgoDBModel: mm,
		d:          d,
		q:          q,
		opts:       opts,
	}
}

type pipeFindDsImpl struct {
	MgoDBModel
	d    MgoAggregate
	q    bson.M
	opts []*options.AggregateOptions
}

func (mm *pipeFindDsImpl) Exec(exec func(i interface{}) error) error {
	return mm.PipeFindAndExec(mm.d, mm.q, exec, mm.opts...)
}

func (mm *pipeFindDsImpl) ExportCSV(w io.Writer, title []string, exec func(writer *csv.Writer, i interface{}) error) error {
	csvWriter := csv.NewWriter(w)
	err := csvWriter.Write(title)
	defer csvWriter.Flush()
	if err != nil {
		return err
	}
	return mm.PipeFindAndExec(mm.d, mm.q, func(i interface{}) error {
		err = exec(csvWriter, i)
		return err
	}, mm.opts...)
}
