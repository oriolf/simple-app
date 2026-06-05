package valueobjects

type EntityVersion uint

func NewEntityVersion() EntityVersion { return EntityVersion(1) }

func (v EntityVersion) Increment() EntityVersion { return EntityVersion(v + 1) }

type Name string

// TODO should be able to «use» validators, extending them, without validators
// knowing about all the existing value objects
func NewName(s string) Name {
	return Name(s)
}

type DNI string

func NewDNI(s string) DNI {
	return DNI(s)
}
