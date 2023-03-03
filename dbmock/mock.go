package e2e

import (
	"fmt"
	"strings"

	"github.com/pubgo/sqlmock"
	"github.com/tidwall/match"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func New(tb TestingTB) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
		expectedSQL = strings.TrimSpace(strings.ReplaceAll(expectedSQL, "**", "*"))
		actualSQL = strings.TrimSpace(strings.ReplaceAll(actualSQL, "  ", " "))

		actualSQLUpper := strings.ToUpper(actualSQL)
		expectedSQLUpper := strings.ToUpper(expectedSQL)
		if actualSQLUpper == expectedSQLUpper || match.Match(actualSQLUpper, expectedSQLUpper) {
			return nil
		}

		tb.Logf("sql not match\n expectedSQL => %s \n actualSQL   => %s \n matchSQL    => %v",
			expectedSQL, actualSQL, match.Match(actualSQLUpper, expectedSQLUpper))

		return fmt.Errorf(`could not match actual sql: "%s" with expected regexp "%s"`, actualSQL, expectedSQL)
	})))

	if err != nil {
		tb.Fatalf("failed to create sql mock, err=%w", err)
		return nil, nil
	}

	tb.Cleanup(func() {
		err = mock.ExpectationsWereMet()
		if err != nil {
			tb.Fatalf("failed to ExpectationsWereMet, err=%w", err)
		}
	})

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 db,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		tb.Fatalf("failed to create gorm, err=%w", err)
		return nil, nil
	}

	return gormDB, mock
}
