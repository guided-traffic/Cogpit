package kinds

// ParseCRDUnguarded is ParseCRD without its recover, for tests that must see a panic.
var ParseCRDUnguarded = parseCRD
