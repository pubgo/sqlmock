package sqlmock

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"
)

type CustomConverter struct{}

func (s CustomConverter) ConvertValue(v interface{}) (driver.Value, error) {
	switch v := v.(type) {
	case string:
		return v, nil
	case []string:
		return v, nil
	case int:
		return v, nil
	default:
		return nil, fmt.Errorf("cannot convert %T with value %v", v, v)
	}
}

func ExampleExpectedExec() {
	db, mock, _ := New()
	result := NewErrorResult(fmt.Errorf("some error"))
	mock.ExpectSql(nil, "^INSERT (.+)").WillReturnResult(result)
	res, _ := db.Exec("INSERT something")
	_, err := res.LastInsertId()
	fmt.Println(err)
	// Output: some error
}

func TestBuildQuery(t *testing.T) {
	db, mock, _ := New()
	query := `
		SELECT
			name,
			email,
			address,
			anotherfield
		FROM user
		where
			name    = 'John'
			and
			address = 'Jakarta'

	`

	mock.ExpectSql(nil, query)
	mock.ExpectPrepare(query)

	db.QueryRow(query)
	db.Exec(query)
	db.Prepare(query)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCustomValueConverterQueryScan(t *testing.T) {
	db, mock, _ := New(ValueConverterOption(CustomConverter{}))
	query := `
		SELECT
			name,
			email,
			address,
			anotherfield
		FROM user
		where
			name    = 'John'
			and
			address = 'Jakarta'

	`
	expectedStringValue := "ValueOne"
	expectedIntValue := 2
	expectedArrayValue := []string{"Three", "Four"}
	mock.ExpectSql(nil, query).WillReturnRows(mock.NewRows([]string{"One", "Two", "Three"}).AddRow(expectedStringValue, expectedIntValue, []string{"Three", "Four"}))
	row := db.QueryRow(query)
	var stringValue string
	var intValue int
	var arrayValue []string
	if e := row.Scan(&stringValue, &intValue, &arrayValue); e != nil {
		t.Error(e)
	}
	if stringValue != expectedStringValue {
		t.Errorf("Expectation %s does not met: %s", expectedStringValue, stringValue)
	}
	if intValue != expectedIntValue {
		t.Errorf("Expectation %d does not met: %d", expectedIntValue, intValue)
	}
	if !reflect.DeepEqual(expectedArrayValue, arrayValue) {
		t.Errorf("Expectation %v does not met: %v", expectedArrayValue, arrayValue)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
