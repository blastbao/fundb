%{

package parser

import (
    "github.com/senarukana/fundb/protocol"
    "strconv"
)

var ParsedQuery *Query

type Token struct {
    Pos    int
    Src    string
}

func (t Token) String() string {
    return t.Src
}

%}

%union {
    sql         *Query
    ident       string
    literal     *protocol.FieldValue
    insert_sql  *InsertQuery
    select_statement *SelectQuery
    selection   *SelectExpression
    column_list *ColumnFields
    value_list  *ValueList
    value_items *ValueItems
    table_exp   *TableExpression
    from_exp    *FromExpression
    where_exp   *WhereExpression
    order_by_exp *OrderByList
    ordering_spec *OrderBy
    scalar      *Scalar
    scalar_list *ScalarList
    int_exp     int
    bool_exp    bool 
    tok         Token
} 

%left OR
%left AND
%left NOT
%left EQUAL GREATER GREATEREQ SMALLER SMALLEREQ /* = <> < > <= >= */
%left PLUS MINUS
%left STAR DIV
%nonassoc UMINUS

%token <tok> LP RP DOT COMMA STAR NULLX 
%token <tok> PLUS MINUS DIV OR AND
%token <tok> SELECT UPDATE DELETE INSERT
%token <tok> INTO VALUES WHERE FROM 
%token <tok> ORDER BY DISTINCT ASC DESC LIMIT
%token <tok> IDENT STRING DOUBLE INT BOOL
%token <tok> EQUAL GREATER GREATEREQ SMALLER SMALLEREQ

%type <sql> sql manipulative_statement
%type <insert_sql> insert_statement
%type <select_statement> select_statement
%type <ident> table column
%type <literal> insert_atom literal
%type <column_list> opt_column_commalist column_commalist
%type <value_list> values_list
%type <value_items> value_items insert_atom_commalist
%type <column_list> opt_column_commalist column_commalist
%type <selection> selection
%type <scalar_list> scalar_exp_commalist
%type <scalar> scalar_exp
%type <table_exp> table_exp
%type <from_exp> from_exp table_ref_commalist
%type <where_exp> opt_where_exp where_exp search_condition predicate comparison_predicate
%type <order_by_exp> opt_order_by_exp ordering_spec_commalist
%type <ordering_spec> ordering_spec
%type <int_exp> opt_asc_desc opt_limit_exp
%type <bool_exp> opt_distinct


%start sql

%%
sql: manipulative_statement {
        ParsedQuery = $1
    }
    

manipulative_statement:
        insert_statement {
            $$ = &Query{ QUERY_INSERT, $1}
        }
    |   select_statement {
            $$ = &Query{ QUERY_SELECT, $1}
        }

select_statement:
        SELECT opt_distinct selection table_exp opt_limit_exp{
            $$ = &SelectQuery{$2, $3, $4, $5}
        }
    ;

opt_distinct:
        /* empty */ {
            $$ = false
        }
    |   DISTINCT {
            $$ = true
        }
    ;

selection:
        STAR {
            $$ = &SelectExpression{true, nil}
        }
    |   scalar_exp_commalist {
            $$ = &SelectExpression{false , $1}
        }

scalar_exp_commalist:
        scalar_exp {
            $$ = NewScalarList($1)
        }
    |   scalar_exp_commalist COMMA scalar_exp {
            $$ = ScalarListAppend($1, $3)
        }

scalar_exp:
        column {
            $$ = &Scalar{SCALAR_IDENT, $1}
        }
    |   literal {
            $$ = &Scalar{SCLAR_LITERAL, $1}
        } 

table_exp: 
        from_exp opt_where_exp opt_order_by_exp {
            $$ = &TableExpression{$1, $2, $3}
        }

from_exp:
        FROM table_ref_commalist {
            $$ = $2
        }

table_ref_commalist:
        table {
            $$ = &FromExpression{$1}
        }

opt_where_exp:
        /* empty */ {
            $$ = nil
        }
    |   where_exp

where_exp:
        WHERE search_condition {
            $$ = $2
        }

search_condition:
        LP search_condition RP {
            $$ = $2
        }
    |   search_condition OR search_condition {
            $$ = &WhereExpression{$1, $3, false, $2.Src}

        }
    |   search_condition AND search_condition {
            $$ = &WhereExpression{$1, $3, false, $2.Src}
        }
    |   predicate

predicate:
        comparison_predicate 

comparison_predicate:
        scalar_exp EQUAL scalar_exp {
            $$ = &WhereExpression{$1, $3, true, $2.Src}
        }
    |   scalar_exp SMALLER scalar_exp {
            $$ = &WhereExpression{$1, $3, true, $2.Src}
        }
    |   scalar_exp GREATER scalar_exp {
            $$ = &WhereExpression{$1, $3, true, $2.Src}
        }
    |   scalar_exp SMALLEREQ scalar_exp {
            $$ = &WhereExpression{$1, $3, true, $2.Src}
        }
    |   scalar_exp GREATEREQ scalar_exp {
            $$ = &WhereExpression{$1, $3, true, $2.Src}
        }

opt_order_by_exp:
        /* empty */ {
            $$ = nil
        }
    |   ORDER BY ordering_spec_commalist {
            $$ = $3
        }

ordering_spec_commalist:
        ordering_spec {
            $$ = NewOrderByList($1)
        }
    |   ordering_spec_commalist COMMA ordering_spec {
            $$ = OrderByListAppend($1, $3)
        }

ordering_spec:
       column opt_asc_desc {
            $$ = &OrderBy{$1, $2}
       }

opt_asc_desc:
        /* empty */ {
            $$ = 0
        }
    |   ASC {
            $$ = 1
        }
    |   DESC {
            $$ = 2
        }
opt_limit_exp:
        /* empty */ {
            $$ = -1
        }
    |   LIMIT INT {
            val, _ := strconv.ParseInt($2.Src, 10, 64)
            $$ = int(val)
    }

insert_statement:
        INSERT INTO table opt_column_commalist VALUES values_list {
            $$ = &InsertQuery{$3, $4, $6}
        }

opt_column_commalist:
        /* empty */ {
            $$ = nil
        }
    |   LP column_commalist RP {
            $$ = $2
        }

column_commalist:
        column {
            $$ = NewColumnField($1)
        }
    |   column_commalist COMMA column {
            $$ = ColumnFieldsAppend($1, $3)
        }

values_list:
        value_items {
            $$ = NewValueList($1)
        }
    |   values_list COMMA value_items {
            $$ = ValueListAppend($1, $3)
        }

value_items:
        LP insert_atom_commalist RP {
            $$ = $2
        }

insert_atom_commalist:
        insert_atom {
            $$ = NewValueItem($1)
        }
    |   insert_atom_commalist COMMA insert_atom {
            $$ = ValueItemAppend($1, $3)
        }

insert_atom:
        literal {
            $$ = $1
        }
    |   NULLX {
            $$ = NewFieldValue(protocol.NULL, "")
        }

column:     
        IDENT {
            $$ = $1.Src
        }

table:
        IDENT {
            $$ = $1.Src
        }

literal:
        STRING {
            $$ = NewFieldValue(protocol.STRING, $1.Src)
        }
    |   INT {
            $$ = NewFieldValue(protocol.INT, $1.Src)
        }
    |   DOUBLE {
            $$ = NewFieldValue(protocol.DOUBLE, $1.Src)
        }
    |   BOOL {
            $$ = NewFieldValue(protocol.BOOL, $1.Src)
        }

%%