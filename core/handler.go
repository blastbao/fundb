package core

import (
	"fmt"
	"sync"

	"github.com/senarukana/fundb/engine"
	"github.com/senarukana/fundb/parser"
	"github.com/senarukana/fundb/protocol"
)

const (
	defaultDatabase   = "default"
	reserverdIdColumn = "_id"
)

type EngineHandler struct {
	*engine.EngineManager
	currentSequenceNumber uint32
	sequenceNumberLock    sync.Mutex
}

func NewEngineHandler(engineName, dataPath string) (*EngineHandler, error) {
	engineManager, err := engine.NewEngineManager(engineName, dataPath)
	if err != nil {
		return nil, err
	}
	return &EngineHandler{
		EngineManager: engineManager,
	}, nil
}

func (self *EngineHandler) Query(sql string) *Response {
	query, err := parser.ParseQuery(sql)
	if err != nil {
		return &Response{
			Error: err,
		}
	}
	switch query.Type {
	case parser.QUERY_SCHEMA_TABLE_CREATE:
		return self.craeteTable(query.Query.(*parser.CreateTableQuery))
	case parser.QUERY_INSERT:
		return self.insert(query.Query.(*parser.InsertQuery))
	case parser.QUERY_DELETE:
		return self.delete(query.Query.(*parser.DeleteQuery))
	case parser.QUERY_SELECT:
		return self.fetch(query.Query.(*parser.SelectQuery))
	default:
		panic(fmt.Sprintf("UNKNOWN Query Type %d", query.Type))
	}
	return nil
}

func (self *EngineHandler) validInsertQuery(query *parser.InsertQuery) error {
	if len(query.Values[0].Items) != len(query.Fields) {
		return fmt.Errorf("syntax error: Incompatible fields(%d) and values(%d)",
			len(query.Values[0].Items), len(query.Fields))
	}

	var paramCount = -1
	for valueIndex, valueItems := range query.Values {
		if paramCount == -1 {
			paramCount = len(valueItems.Items)
		}
		if paramCount != len(valueItems.Items) {
			return fmt.Errorf("syntax error: Incompatible value paramters in %d, paremter num is %d, exptected %d",
				valueIndex, len(valueItems.Items), paramCount)
		}

	}
	return nil
}

func (self *EngineHandler) craeteTable(query *parser.CreateTableQuery) *Response {
	return &Response{
		Error: self.CreateTable(query.Name, query.Type),
	}
}

// append _id column in both fields and values
func (self *EngineHandler) appendIdIfNeeded(query *parser.InsertQuery) {
	foundId := false
	for _, field := range query.Fields {
		if field == reserverdIdColumn {
			foundId = true
			break
		}
	}
	if !foundId {
		query.Fields = append(query.Fields, reserverdIdColumn)
		for _, valueItems := range query.Values {
			dummyIdField := &parser.NullNode{protocol.NULL, new(protocol.FieldValue)}
			valueItems.Items = append(valueItems.Items, dummyIdField)
		}
	}
}

func (self *EngineHandler) insert(query *parser.InsertQuery) *Response {
	err := self.validInsertQuery(query)
	if err == nil {
		self.appendIdIfNeeded(query)
		recordList := &protocol.RecordList{
			Name:   &query.Table,
			Fields: query.Fields,
			Values: make([]*protocol.Record, 0, len(query.Values)),
		}
		self.sequenceNumberLock.Lock()
		defer self.sequenceNumberLock.Unlock()

		for _, valueItems := range query.Values {
			sn := self.currentSequenceNumber
			record := &protocol.Record{
				SequenceNum: &sn,
				Values:      make([]*protocol.FieldValue, 0, len(valueItems.Items)),
			}
			for _, item := range valueItems.Items {
				record.Values = append(record.Values, item.GetVal())
			}
			self.currentSequenceNumber++
			recordList.Values = append(recordList.Values, record)
		}
		err = self.Insert(recordList)
	}

	if err != nil {
		return &Response{
			Error: err,
		}
	} else {
		return &Response{
			RowsAffected: uint64(len(query.ValueList.Values)),
		}
	}
}

func (self *EngineHandler) delete(query *parser.DeleteQuery) *Response {
	rowsAffected, err := self.Delete(query)
	if err != nil {
		return &Response{
			Error: err,
		}
	} else {
		return &Response{
			RowsAffected: uint64(rowsAffected),
		}
	}
}

func (self *EngineHandler) fetch(query *parser.SelectQuery) *Response {
	resultList, err := self.Fetch(query)

	if err != nil {
		return &Response{
			Error: err,
		}
	} else {
		return &Response{
			RowsAffected: uint64(len(resultList.Values)),
			Results:      resultList,
		}
	}
}