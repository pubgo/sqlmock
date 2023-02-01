package e2e

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	"github.com/pubgo/sqlmock"
	"github.com/tidwall/match"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type DbMock struct {
	tb   TestingTB
	mock sqlmock.Sqlmock
	db   *gorm.DB

	query      bool
	delete     bool
	update     bool
	create     bool
	tx         bool
	prepare    bool
	column     []*schema.Field
	tableName  string
	checker    func(args []driver.Value) error
	optChecker sqlmock.Matcher
	model      schema.Tabler
	sql        string
	args       []driver.Value
}

func (m *DbMock) Mock() sqlmock.Sqlmock { return m.mock }
func (m *DbMock) DB() *gorm.DB          { return m.db }

func (m *DbMock) createExpect(model schema.Tabler) *DbMock {
	if model == nil {
		m.tb.Fatalf("model is nil")
		return m
	}

	return &DbMock{
		mock:      m.mock,
		db:        m.db,
		tb:        m.tb,
		model:     model,
		tableName: model.TableName(),
		column:    parseColumn(model),
	}
}

func (m *DbMock) do(err error, ret driver.Result, rows *sqlmock.Rows) {
	if m.tx {
		m.mock.ExpectBegin()
	}

	var sql = ""
	if m.query {
		sql = selectSql(m.tableName, sql)
	}

	if m.create {
		sql = insertSql(m.tableName)
	}

	if m.update {
		sql = updateSql(m.tableName, sql)
	}

	if m.delete {
		sql = deleteSql(m.tableName, sql)
	}

	if m.prepare {
		m.mock.ExpectPrepare(sql)
	}

	if m.sql != "" {
		sql = m.sql
	}

	e := m.mock.ExpectSql(m.optChecker, sql)

	if m.checker != nil {
		e = e.WithArgsCheck(m.checker)
	}

	if m.create {
		var args []driver.Value
		var reflectValue = reflect.ValueOf(m.model)
		for _, name := range m.column {
			if name.PrimaryKey {
				continue
			}

			fv, _ := name.ValueOf(context.Background(), reflectValue)
			args = append(args, fv)
		}
		e = e.WithArgs(args...)
	}

	if m.query || m.delete {
		var args []driver.Value
		var reflectValue = reflect.ValueOf(m.model)
		for _, name := range m.column {
			fv, zero := name.ValueOf(context.Background(), reflectValue)
			if zero {
				continue
			}

			args = append(args, fv)
		}
		e = e.WithArgs(args...)
	}

	if len(m.args) > 0 {
		e = e.WithArgs(m.args...)
	}

	if err != nil {
		e = e.WillReturnError(err)
	}

	if rows != nil {
		e = e.WillReturnRows(rows)
	}

	if ret != nil {
		e.WillReturnResult(ret)
	}

	if m.tx {
		if err == nil {
			m.mock.ExpectCommit()
		} else {
			m.mock.ExpectRollback()
		}
	}
}

func (m *DbMock) WithTx() *DbMock {
	m.tx = true
	return m
}

func (m *DbMock) WithArgs(args ...driver.Value) *DbMock {
	m.args = args
	return m
}

func (m *DbMock) WithPrepare() *DbMock {
	m.prepare = true
	return m
}

func (m *DbMock) WithArgsChecker(checker func(args []driver.Value) error) *DbMock {
	m.checker = checker
	return m
}

// WithOpt Opt=[exec,query]
func (m *DbMock) WithOpt(checker sqlmock.Matcher) *DbMock {
	m.optChecker = checker
	return m
}

func (m *DbMock) ReturnErr(err error) {
	m.do(err, nil, nil)
}

func (m *DbMock) ReturnResult(lastInsertID int64, rowsAffected int64) {
	m.do(nil, sqlmock.NewResult(lastInsertID, rowsAffected), nil)
}

func (m *DbMock) Return(returns interface{}) {
	m.do(nil, nil, ModelToRows(returns))
}

func (m *DbMock) Sql(sql string) *DbMock {
	m.sql = sql
	return m
}

func (m *DbMock) Create(model schema.Tabler) *DbMock {
	var mm = m.createExpect(model)
	mm.create = true
	return mm
}

func (m *DbMock) Delete(model schema.Tabler) *DbMock {
	var mm = m.createExpect(model)
	mm.delete = true
	return mm
}

func (m *DbMock) Update(model schema.Tabler) *DbMock {
	var mm = m.createExpect(model)
	mm.update = true
	return mm
}

func (m *DbMock) Find(model schema.Tabler) *DbMock {
	var mm = m.createExpect(model)
	mm.query = true
	return mm
}

func New(tb TestingTB) *DbMock {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
		expectedSQL = strings.TrimSpace(strings.ReplaceAll(expectedSQL, "**", "*"))
		actualSQL = strings.TrimSpace(strings.ReplaceAll(actualSQL, "  ", " "))

		expectedSQL = strings.ToUpper(expectedSQL)
		actualSQL = strings.ToUpper(actualSQL)
		if actualSQL == expectedSQL || match.Match(actualSQL, expectedSQL) {
			return nil
		}

		tb.Logf("sql not match\n expectedSQL => %s \n actualSQL   => %s \n matchSQL    => %v",
			expectedSQL, actualSQL, match.Match(strings.ToUpper(actualSQL), strings.ToUpper(expectedSQL)))

		return fmt.Errorf(`could not match actual sql: "%s" with expected regexp "%s"`, actualSQL, expectedSQL)
	})))

	if err != nil {
		tb.Fatalf("%v", err)
		return nil
	}

	tb.Cleanup(func() {
		err := mock.ExpectationsWereMet()
		if err != nil {
			tb.Fatalf("%v", err)
		}
	})

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 db,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		//SkipDefaultTransaction: true,
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		tb.Fatalf("%v", err)
		return nil
	}

	return &DbMock{db: gormDB, mock: mock, tb: tb}
}
