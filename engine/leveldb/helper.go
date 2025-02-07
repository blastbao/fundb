package leveldb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"

	"github.com/senarukana/fundb/protocol"
)

type rawRecordValue struct {
	*recordKey
	value []byte
}

type recordKey struct {
	key []byte
}

func newRecordKey(key []byte) *recordKey {
	return &recordKey{
		key: key,
	}
}

func (r *recordKey) getFieldId() []byte {
	return r.key[0:8]
}

func (r *recordKey) getId() []byte {
	return r.key[8:16]
}

func (r *recordKey) getTimestamp() []byte {
	return r.key[16:24]
}

func (r *recordKey) getSequenceNum() []byte {
	return r.key[24:]
}

func getIdFromKey(key []byte) []byte {
	return key[8:16]
}

func genereateColumnId(table, column string) []byte {
	h := fnv.New64()
	b := []byte(fmt.Sprintf("%s%c%s", table, SEPERATOR, column))
	idBuffer := bytes.NewBuffer(make([]byte, 0, 8))
	h.Write(b)
	val := h.Sum64()
	binary.Write(idBuffer, binary.BigEndian, val)
	return idBuffer.Bytes()
}

func genereateMetaTableKey(table string) []byte {
	return []byte(table)
}

func appendReversedIdFieldsIfNeeded(fields []string) []string {
	for _, field := range fields {
		if field == RESERVED_ID_COLUMN {
			return fields
		}
	}
	return append(fields, RESERVED_ID_COLUMN)
}

func getIdsFromRecords(fields []string, records []*protocol.Record) (res []int64) {
	res = make([]int64, 0, len(records))
	idIdx := -1
	for i, field := range fields {
		if field == RESERVED_ID_COLUMN {
			idIdx = i
		}
	}

	if idIdx == -1 {
		panic(fmt.Sprintf("%s NOT FOUND in record list", RESERVED_ID_COLUMN))
	}

	for _, record := range records {
		res = append(res, record.Values[idIdx].GetIntVal())
	}
	return res
}
