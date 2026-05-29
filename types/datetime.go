package types

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

type DateTime struct {
	time.Time
}

func Now() DateTime                              { return DateTime{time.Now()} }
func (dt DateTime) Add(d time.Duration) DateTime { return DateTime{dt.Time.Add(d)} }

func (d DateTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

func (d *DateTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if err := d.FromString(s); err != nil {
		return fmt.Errorf("could not scan date time: %w", err)
	}

	return nil
}

func (d DateTime) Value() (driver.Value, error) {
	return d.String(), nil
}

func (d *DateTime) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("must receive a non-null string")
	}
	switch value := value.(type) {
	case string:
		return d.FromString(value)
	case []byte:
		return d.FromString(string(value))
	}

	return fmt.Errorf("must receive a string")
}

func (d DateTime) String() string {
	return d.Format(time.RFC3339Nano)
}

func (d *DateTime) FromString(s string) (err error) {
	d.Time, err = time.Parse(time.RFC3339Nano, s)
	return err
}
