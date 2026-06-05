package types

import (
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

type UUID [16]byte

func NewUUID() UUID {
	var uuid UUID
	_, err := io.ReadFull(rand.Reader, uuid[:])
	if err != nil {
		panic(fmt.Sprintf("Cannot read from crypto/rand: %s", err))
	}

	// version 4, variant 2
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return uuid
}

func (uuid UUID) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, uuid.String())), nil
}

func (uuid *UUID) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if err := uuid.FromString(s); err != nil {
		return fmt.Errorf("could not scan uuid: %w", err)
	}

	return nil
}

func (uuid UUID) Value() (driver.Value, error) {
	return uuid.String(), nil
}

func (uuid *UUID) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("must receive a non-null string")
	}
	switch value := value.(type) {
	case string:
		return uuid.FromString(value)
	case []byte:
		return uuid.FromString(string(value))
	}

	return fmt.Errorf("must receive a string")
}

func ScanUUID(rows *sql.Rows) (UUID, error) {
	var uuid UUID
	return uuid, rows.Scan(&uuid)
}

func (uuid UUID) String() string {
	s := [32]byte{}
	hex.Encode(s[:], uuid[:])

	out := make([]byte, 0, 36)
	for _, lh := range [][2]int{{0, 8}, {8, 12}, {12, 16}, {16, 20}} {
		out = append(out, s[lh[0]:lh[1]]...)
		out = append(out, '-')
	}
	out = append(out, s[20:32]...)
	return string(out)
}

func (uuid *UUID) FromString(s string) error {
	if len(s) != 36 {
		return errors.New("string must have 36 characters")
	}

	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return errors.New("string must have four dashes")
	}

	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 32 {
		return errors.New("string without dashes must have 32 characters")
	}

	if _, err := hex.Decode((*uuid)[:], []byte(s)); err != nil {
		return fmt.Errorf("could not decode hexadecimal: %w", err)
	}

	return nil
}
