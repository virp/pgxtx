//revive:disable:exported
package mocks

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Rows struct {
	t testingT

	data              [][]any
	err               error
	commandTag        pgconn.CommandTag
	fieldDescriptions []pgconn.FieldDescription
	columnCount       int

	idx    int
	closed bool
}

func NewRows(t testingT) *Rows {
	return &Rows{t: t}
}

func (r *Rows) AddRow(values ...any) *Rows {
	if len(r.data) == 0 {
		r.columnCount = len(values)
	} else if len(values) != r.columnCount {
		panic("mocks.Rows.AddRow: inconsistent column count")
	}

	row := make([]any, len(values))
	copy(row, values)
	r.data = append(r.data, row)
	return r
}

func (r *Rows) WithError(err error) *Rows {
	r.err = err
	return r
}

func (r *Rows) WithCommandTag(tag pgconn.CommandTag) *Rows {
	r.commandTag = tag
	return r
}

func (r *Rows) WithFieldDescriptions(fields []pgconn.FieldDescription) *Rows {
	r.fieldDescriptions = append([]pgconn.FieldDescription(nil), fields...)
	return r
}

func (r *Rows) Close() {
	r.closed = true
}

func (r *Rows) Err() error {
	return r.err
}

func (r *Rows) CommandTag() pgconn.CommandTag {
	return r.commandTag
}

func (r *Rows) FieldDescriptions() []pgconn.FieldDescription {
	return append([]pgconn.FieldDescription(nil), r.fieldDescriptions...)
}

func (r *Rows) Next() bool {
	if r.closed {
		return false
	}
	if r.idx >= len(r.data) {
		r.closed = true
		return false
	}
	r.idx++
	return true
}

func (r *Rows) Scan(dest ...any) error {
	row, err := r.currentRow()
	if err != nil {
		return err
	}
	if len(dest) != len(row) {
		return errors.New("scan dest count does not match row values")
	}
	for i := range dest {
		if err := assignValue(dest[i], row[i]); err != nil {
			return fmt.Errorf("scan column %d: %w", i, err)
		}
	}
	return nil
}

func (r *Rows) Values() ([]any, error) {
	row, err := r.currentRow()
	if err != nil {
		return nil, err
	}
	values := make([]any, len(row))
	copy(values, row)
	return values, nil
}

func (r *Rows) RawValues() [][]byte {
	return nil
}

func (r *Rows) Conn() *pgx.Conn {
	return nil
}

func (r *Rows) currentRow() ([]any, error) {
	if r.idx == 0 {
		return nil, errors.New("row is not positioned, call Next first")
	}
	if r.closed {
		return nil, errors.New("row is not available")
	}
	if r.idx > len(r.data) {
		return nil, errors.New("row is not available")
	}
	return r.data[r.idx-1], nil
}

func assignValue(dest, src any) error {
	if dest == nil {
		return nil
	}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
		return errors.New("scan destination must be a non-nil pointer")
	}

	target := destValue.Elem()
	if src == nil {
		if target.Kind() == reflect.Ptr {
			target.Set(reflect.Zero(target.Type()))
			return nil
		}
		return fmt.Errorf("cannot scan NULL into %T", dest)
	}

	srcValue := reflect.ValueOf(src)
	if srcValue.Type().AssignableTo(target.Type()) {
		target.Set(srcValue)
		return nil
	}
	if srcValue.Type().ConvertibleTo(target.Type()) {
		target.Set(srcValue.Convert(target.Type()))
		return nil
	}

	return errors.New("scan destination type does not match row value")
}

//revive:enable:exported
