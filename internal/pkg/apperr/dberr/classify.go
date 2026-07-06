package dberr

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type DBErrorKind int

// 상태를 미식별, 재시도 가능, 제약조건, 데이터 불일치, 불명확으로 나눈다.
const (
	DBErrorUnknown DBErrorKind = iota
	DBErrorRetryable
	DBErrorConstraint
	DBErrorNotFound
	DBErrorAmbiguous
)

func ClassifyDBError(err error) DBErrorKind {
	switch {
	case errors.Is(err, context.DeadlineExceeded),
		errors.Is(err, context.Canceled),
		errors.Is(err, driver.ErrBadConn),
		errors.Is(err, sql.ErrConnDone):
		return DBErrorAmbiguous
	case errors.Is(err, ErrNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return DBErrorNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey),
		errors.Is(err, gorm.ErrForeignKeyViolated):
		return DBErrorConstraint
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number { // 데드락 또는 timeout은 재시도 가능으로
		case 1205, 1213:
			return DBErrorRetryable
		case 1062, 1048, 1406, 1451, 1452: // 제약 조건 위반 문제
			return DBErrorConstraint
		}
	}

	return DBErrorUnknown
}
