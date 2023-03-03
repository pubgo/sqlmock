package e2e

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"sync"

	"github.com/pubgo/sqlmock"
	"gorm.io/gorm/schema"
)

func NewRows(columns ...string) *sqlmock.Rows {
	return sqlmock.NewRows(columns)
}

func ModelToRows[T any](objs ...*T) *sqlmock.Rows {
	var t T
	columns, _ := getColumns(&t)
	rows := sqlmock.NewRows(columns)
	for _, w := range objs {
		values, _ := getValues(w, columns)
		rows.AddRow(values...)
	}
	return rows
}

func getValues(dest any, columns []string) ([]driver.Value, error) {
	s, err := schema.Parse(dest, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return nil, err
	}

	rv := reflect.ValueOf(dest)
	values := make([]driver.Value, 0, len(columns))
	for _, col := range columns {
		fv, _ := s.FieldsByDBName[col].ValueOf(context.Background(), rv)
		values = append(values, fv)
	}
	return values, nil
}

func getColumns(dest any) ([]string, error) {
	s, err := schema.Parse(dest, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		return nil, err
	}

	columns := make([]string, 0, len(s.Fields))
	for _, v := range s.Fields {
		if len(v.DBName) != 0 {
			columns = append(columns, v.DBName)
		}
	}
	return columns, nil
}

func insertSql(tableName string) string {
	return "INSERT INTO" + fmt.Sprintf(` "%s" *`, tableName)
}

func deleteSql(tableName string, where string) string {
	if where == "" {
		return "DELETE FROM" + fmt.Sprintf(` "%s"*`, tableName)
	}
	return "DELETE FROM" + fmt.Sprintf(` "%s" WHERE %s*`, tableName, where)
}

func updateSql(tableName string, where string) string {
	if where == "" {
		return "UPDATE" + fmt.Sprintf(` "%s" SET*`, tableName)
	}
	return "UPDATE" + fmt.Sprintf(` "%s" SET * WHERE %s*`, tableName, where)
}

func selectSql(tableName string, where string) string {
	if where == "" {
		return "SELECT * FROM" + fmt.Sprintf(` "%s"*`, tableName)
	}
	return "SELECT * FROM" + fmt.Sprintf(` "%s" WHERE %s*`, tableName, where)
}

func parseVal(val interface{}) []driver.Value { //nolint
	var values []driver.Value
	if val == nil {
		values = append(values, nil)
		return values
	}

	var vv = reflect.ValueOf(val)
	for vv.Kind() == reflect.Ptr {
		vv = vv.Elem()
	}

	if vv.Kind() == reflect.Array || vv.Kind() == reflect.Slice {
		for i := 0; i < vv.Len(); i++ {
			values = append(values, vv.Index(i).Interface())
		}
	} else {
		values = append(values, val)
	}
	return values
}
