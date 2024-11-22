// internal/models/schema.go
package models

import _ "embed"

//go:embed schema.sql
var SchemaSQL string
