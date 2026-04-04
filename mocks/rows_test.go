package mocks

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRowsIteratesAndScans(t *testing.T) {
	rows := NewRows(t).
		AddRow(int64(1), "John").
		AddRow(int64(2), "Jane")

	require.True(t, rows.Next())

	var id int64
	var name string
	require.NoError(t, rows.Scan(&id, &name))
	assert.Equal(t, int64(1), id)
	assert.Equal(t, "John", name)

	require.True(t, rows.Next())
	require.NoError(t, rows.Scan(&id, &name))
	assert.Equal(t, int64(2), id)
	assert.Equal(t, "Jane", name)

	assert.False(t, rows.Next())
	assert.NoError(t, rows.Err())
}

func TestRowsValuesReturnsCurrentRow(t *testing.T) {
	rows := NewRows(t).
		AddRow(int64(1), "John")

	require.True(t, rows.Next())

	values, err := rows.Values()
	require.NoError(t, err)
	assert.Equal(t, []any{int64(1), "John"}, values)

	values[1] = "Changed"

	valuesAgain, err := rows.Values()
	require.NoError(t, err)
	assert.Equal(t, []any{int64(1), "John"}, valuesAgain)
}

func TestRowsErrReturnsConfiguredError(t *testing.T) {
	expectedErr := errors.New("rows error")
	rows := NewRows(t).
		AddRow(int64(1)).
		WithError(expectedErr)

	require.True(t, rows.Next())
	assert.False(t, rows.Next())
	assert.ErrorIs(t, rows.Err(), expectedErr)
}

func TestRowsCommandTagAndFieldDescriptions(t *testing.T) {
	tag := pgconn.NewCommandTag("SELECT 2")
	fields := []pgconn.FieldDescription{{Name: "id"}, {Name: "name"}}

	rows := NewRows(t).
		WithCommandTag(tag).
		WithFieldDescriptions(fields)

	assert.Equal(t, tag, rows.CommandTag())
	assert.Equal(t, fields, rows.FieldDescriptions())

	returnedFields := rows.FieldDescriptions()
	returnedFields[0].Name = "changed"
	assert.Equal(t, fields, rows.FieldDescriptions())
}

func TestRowsScanWithoutCurrentRowReturnsError(t *testing.T) {
	rows := NewRows(t).
		AddRow(int64(1), "John")

	var id int64
	var name string
	err := rows.Scan(&id, &name)
	require.Error(t, err)
	assert.Equal(t, "row is not positioned, call Next first", err.Error())
}

func TestRowsValuesWithoutCurrentRowReturnsError(t *testing.T) {
	rows := NewRows(t).
		AddRow(int64(1), "John")

	_, err := rows.Values()
	require.Error(t, err)
	assert.Equal(t, "row is not positioned, call Next first", err.Error())
}

func TestRowsScanChecksDestinationCount(t *testing.T) {
	rows := NewRows(t).
		AddRow(int64(1), "John")

	require.True(t, rows.Next())

	var id int64
	err := rows.Scan(&id)
	require.Error(t, err)
	assert.Equal(t, "scan dest count does not match row values", err.Error())
}

func TestRowsScanRequiresNonNilPointerDestinations(t *testing.T) {
	rows := NewRows(t).
		AddRow(int64(1), "John")

	require.True(t, rows.Next())

	var id int64
	require.NoError(t, rows.Scan(&id, nil))
}

func TestRowsScanRejectsNonPointerDestination(t *testing.T) {
	rows := NewRows(t).
		AddRow(int64(1))

	require.True(t, rows.Next())

	var id int64
	err := rows.Scan(id)
	require.Error(t, err)
	assert.Equal(t, "scan column 0: scan destination must be a non-nil pointer", err.Error())
}

func TestRowsScanRejectsIncompatibleDestinationType(t *testing.T) {
	rows := NewRows(t).
		AddRow("John")

	require.True(t, rows.Next())

	var id int64
	err := rows.Scan(&id)
	require.Error(t, err)
	assert.Equal(t, "scan column 0: scan destination type does not match row value", err.Error())
}

func TestRowsScanRejectsNilSourceForSinglePointer(t *testing.T) {
	rows := NewRows(t).
		AddRow(nil)

	require.True(t, rows.Next())

	var name string
	err := rows.Scan(&name)
	require.Error(t, err)
	assert.Equal(t, "scan column 0: cannot scan NULL into *string", err.Error())
}

func TestRowsScanAssignsNilForPointerToPointer(t *testing.T) {
	rows := NewRows(t).
		AddRow(nil)

	require.True(t, rows.Next())

	name := new("John")
	require.NoError(t, rows.Scan(&name))
	assert.Nil(t, name)
}

func TestRowsCloseStopsIteration(t *testing.T) {
	rows := NewRows(t).
		AddRow(int64(1), "John")

	rows.Close()

	assert.False(t, rows.Next())
}

func TestRowsValuesAfterExhaustionReturnsError(t *testing.T) {
	rows := NewRows(t).
		AddRow(int64(1), "John")

	require.True(t, rows.Next())
	assert.False(t, rows.Next())

	_, err := rows.Values()
	require.Error(t, err)
	assert.Equal(t, "row is not available", err.Error())
}

func TestRowsAddRowPanicsOnInconsistentColumnCount(t *testing.T) {
	rows := NewRows(t).
		AddRow(int64(1), "John")

	require.PanicsWithValue(t, "mocks.Rows.AddRow: inconsistent column count", func() {
		rows.AddRow(int64(2))
	})
}
