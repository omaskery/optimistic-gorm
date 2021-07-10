package optimistic

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrConcurrentModification is returned when concurrent modification is detected during an Update or Delete operation
// on a Versioned model
var ErrConcurrentModification = errors.New("concurrent modification detected")

// Versioned can be embedded in a GORM model to add optimistic locking
type Versioned struct {
	Version uint64 `gorm:"not null;default:1;"`
}

// BeforeUpdate ensures that updates to a Versioned model only apply if there has not been a concurrent modification,
// detected through an optimistic lock version, and asserts that the new object will have a new version
func (v *Versioned) BeforeUpdate(tx *gorm.DB) error {
	return v.assertLockValidity(tx)
}

// AfterUpdate detects concurrent modification issues
func (v *Versioned) AfterUpdate(tx *gorm.DB) error {
	return v.ensureRowsAffected(tx)
}

// BeforeDelete ensures that deleting a Versioned model only applies if there has not been a concurrent modification,
// detected through an optimistic lock version, and asserts that the deleted object will have a new version
func (v *Versioned) BeforeDelete(tx *gorm.DB) error {
	return v.assertLockValidity(tx)
}

// AfterDelete detects concurrent modification issues
func (v *Versioned) AfterDelete(tx *gorm.DB) error {
	return v.ensureRowsAffected(tx)
}

func (v *Versioned) assertLockValidity(tx *gorm.DB) error {
	readVersion := v.Version
	v.Version = readVersion + 1

	tx.Statement.Where("version = ?", readVersion)
	tx.Statement.AddClause(clause.Set{{Column: clause.Column{Name: "version"}, Value: v.Version}})

	return nil
}

func (v *Versioned) ensureRowsAffected(tx *gorm.DB) error {
	if tx.Statement.DB.RowsAffected < 1 {
		return ErrConcurrentModification
	}

	return nil
}
