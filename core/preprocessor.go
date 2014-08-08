package core

import (
	"fmt"
	"math"

	"github.com/senarukana/fundb/parser"
	"github.com/senarukana/fundb/protocol"
	"github.com/senarukana/fundb/util"
)

const (
	InvalidRange int64 = math.MaxInt64
)

func getIdFromBetween(condition *parser.WhereExpression) (int64, int64, error) {
	field := condition.Left.(string)
	if field != "_id" {
		return InvalidRange, InvalidRange, nil
	}
	betweenExpr := condition.Right.(*parser.BetweenExpression)
	if betweenExpr.Left.Type != parser.SCLAR_LITERAL || betweenExpr.Right.Type != parser.SCLAR_LITERAL {
		panic("NOT SUPPORTED SCALAR TYPE")
	}
	leftField := betweenExpr.Left.Val.(parser.LiteralNode)
	rightField := betweenExpr.Right.Val.(parser.LiteralNode)

	if leftField.GetType() != protocol.INT || rightField.GetType() != protocol.INT {
		return InvalidRange, InvalidRange, fmt.Errorf("Invalid _id type %v, exptected INT", leftField.GetType())
	}
	idStart := leftField.GetVal().GetIntVal()
	idEnd := rightField.GetVal().GetIntVal()
	if idStart > idEnd {
		return InvalidRange, InvalidRange, fmt.Errorf("Range of Between is invalid, %d is bigger than %d", idStart, idEnd)
	}
	return idStart, idEnd - 1, nil
}

func getIdFromComparison(condition *parser.WhereExpression) (int64, int64, error) {
	fieldName := condition.Left.(string)
	if fieldName != "_id" {
		return InvalidRange, InvalidRange, nil
	}
	rightScalar := condition.Right.(*parser.Scalar)
	if rightScalar.Type != parser.SCLAR_LITERAL {
		panic("NOT SUPPORTED SCALAR TYPE")
	}
	rightNode := rightScalar.Val.(parser.LiteralNode)
	if rightNode.GetType() != protocol.INT {
		return InvalidRange, InvalidRange, fmt.Errorf("Invalid _id type %v, exptected INT", rightNode.GetType())
	}
	val := rightNode.GetVal().GetIntVal()
	switch parser.ComparisonMap[condition.Token.Src] {
	case parser.EQUAL:
		return val, val, nil
	case parser.GREATER:
		return val + 1, InvalidRange, nil
	case parser.GREATEREQ:
		return val, InvalidRange, nil
	case parser.SMALLER:
		return InvalidRange, val - 1, nil
	case parser.SMALLEREQ:
		return InvalidRange, val, nil
	default:
		panic("Invalid token type")
	}
	panic("shouldn't go here")
}

func getIdCondition(condition *parser.WhereExpression) (*parser.WhereExpression, int64, int64, error) {
	if condition == nil {
		return nil, InvalidRange, InvalidRange, nil
	}
	switch condition.Type {
	case parser.WHERE_BETWEEN:
		idStart, idEnd, err := getIdFromBetween(condition)
		return nil, idStart, idEnd, err
	case parser.WHERE_COMPARISON:
		idStart, idEnd, err := getIdFromComparison(condition)
		return nil, idStart, idEnd, err
	case parser.WHERE_AND:
		leftCondition, leftStart, leftEnd, err := getIdCondition(condition.Left.(*parser.WhereExpression))
		if err != nil {
			return nil, InvalidRange, InvalidRange, err
		}
		rightCondition, rightStart, rightEnd, err := getIdCondition(condition.Right.(*parser.WhereExpression))
		if err != nil {
			return nil, InvalidRange, InvalidRange, err
		}
		newCondition := condition
		if leftCondition == nil {
			newCondition = rightCondition
		} else if rightCondition == nil {
			newCondition = leftCondition
		} else {
			newCondition.Left = leftCondition
			newCondition.Right = rightCondition
		}
		var idStart, idEnd int64
		if leftStart == InvalidRange && rightStart == InvalidRange {
			idStart = 0
		} else if leftStart != InvalidRange && rightStart == InvalidRange {
			idStart = leftStart
		} else if rightStart != InvalidRange && leftStart == InvalidRange {
			idStart = rightStart
		} else {
			if leftStart > rightStart {
				idStart = leftStart
			} else {
				idStart = rightStart
			}
		}

		if leftEnd == InvalidRange && rightEnd == InvalidRange {
			idEnd = InvalidRange
		} else if leftEnd != InvalidRange && rightEnd == InvalidRange {
			idEnd = leftEnd
		} else if rightEnd != InvalidRange && leftEnd == InvalidRange {
			idEnd = rightEnd
		} else {
			if leftEnd < rightEnd {
				idEnd = leftEnd
			} else {
				idEnd = rightEnd
			}
		}

		return newCondition, idStart, idEnd, nil
	}
	panic("shouldn't go here")
}

func getSelectionColumns(scalarList []*parser.Scalar, columnSet util.StringSet) {
	for _, scalar := range scalarList {
		switch scalar.Type {
		case parser.SCALAR_IDENT:
			ident := scalar.Val.(string)
			columnSet.Insert(ident)
		default:
			panic("SCALAR TYPE NOT SUPPORTED")
		}
	}
}

func getWhereColumns(condition *parser.WhereExpression, columnSet util.StringSet) {
	if condition == nil {
		return
	}
	switch condition.Type {
	case parser.WHERE_AND:
		getWhereColumns(condition.Left.(*parser.WhereExpression), columnSet)
		getWhereColumns(condition.Right.(*parser.WhereExpression), columnSet)
	case parser.WHERE_COMPARISON:
		fieldName := condition.Left.(string)
		columnSet.Insert(fieldName)
	case parser.WHERE_BETWEEN:
		fieldName := condition.Left.(string)
		columnSet.Insert(fieldName)
	default:
		panic(fmt.Sprintf("UNKNOWN WHERE TYPE %d", condition.Type))
	}
}

func getFetchColumns(query *parser.SelectQuery) ([]string, []string) {
	columnSet := util.NewStringSet()
	getSelectionColumns(query.ScalarList.ScalarList, columnSet)
	selectColumns := columnSet.ConvertToStrings()
	getWhereColumns(query.WhereExpression, columnSet)
	return selectColumns, columnSet.ConvertToStrings()
}