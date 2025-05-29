package postgres

import (
	"database/sql"
	"reflect"

	_ "github.com/lib/pq"
)

// NullString is an alias for sql.NullString data type
type NullString sql.NullString

// NewNullString creates a new NullString with the given value
func NewNullString(value string) NullString {
	return NullString{String: value, Valid: true}
}

// NewNullStringFromPtr creates a NullString from a string pointer
// If the pointer is nil, Valid will be false
func NewNullStringFromPtr(ptr *string) NullString {
	if ptr == nil {
		return NullString{String: "", Valid: false}
	}
	return NullString{String: *ptr, Valid: true}
}

// Scan implements the Scanner interface for NullString
func (ns *NullString) Scan(value interface{}) error {
	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}

	// if nil then make Valid false
	if reflect.TypeOf(value) == nil {
		*ns = NullString{s.String, false}
	} else {
		*ns = NullString{s.String, true}
	}

	return nil
}

func (ns NullString) ToPtr() *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// NullInt is an alias for sql.NullInt64 data type
type NullInt sql.NullInt64

// NewNullInt creates a new NullInt with the given value
func NewNullInt(value int64) NullInt {
	return NullInt{Int64: value, Valid: true}
}

// NewNullIntFromPtr creates a NullInt from an int64 pointer
// If the pointer is nil, Valid will be false
func NewNullIntFromPtr(ptr *int64) NullInt {
	if ptr == nil {
		return NullInt{Int64: 0, Valid: false}
	}
	return NullInt{Int64: *ptr, Valid: true}
}

// Scan implements the Scanner interface for NullInt
func (ni *NullInt) Scan(value interface{}) error {
	var i sql.NullInt64
	if err := i.Scan(value); err != nil {
		return err
	}

	// if nil then make Valid false
	if reflect.TypeOf(value) == nil {
		*ni = NullInt{i.Int64, false}
	} else {
		*ni = NullInt{i.Int64, true}
	}

	return nil
}

func (ni NullInt) ToPtr() *int64 {
	if !ni.Valid {
		return nil
	}
	return &ni.Int64
}
