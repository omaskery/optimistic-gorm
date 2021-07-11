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
	Version     uint64 `gorm:"not null;default:1;"`
	readVersion uint64 `gorm:"-"`
}

// BeforeUpdate ensures that updates to a Versioned model only apply if there has not been a concurrent modification,
// detected through an optimistic lock version, and asserts that the new object will have a new version
func (v *Versioned) BeforeUpdate(tx *gorm.DB) error {
	return v.assertLockValidity(tx, true)
}

// AfterUpdate detects concurrent modification issues
func (v *Versioned) AfterUpdate(tx *gorm.DB) error {
	return v.ensureRowsAffected(tx)
}

// BeforeDelete ensures that deleting a Versioned model only applies if there has not been a concurrent modification,
// detected through an optimistic lock version, and asserts that the deleted object will have a new version
func (v *Versioned) BeforeDelete(tx *gorm.DB) error {
	isSoftDelete := !tx.Statement.Unscoped
	return v.assertLockValidity(tx, isSoftDelete)
}

// AfterDelete detects concurrent modification issues
func (v *Versioned) AfterDelete(tx *gorm.DB) error {
	if err := v.ensureRowsAffected(tx); err != nil {
		return err
	}

	if tx.Error != nil {
		return nil
	}

	// workaround for GORM issue https://github.com/go-gorm/gorm/pull/3893#issuecomment-877706731
	isSoftDelete := !tx.Statement.Unscoped
	if isSoftDelete {
		tx.Unscoped().Model(tx.Statement.Dest).Where("version = ?", v.readVersion).UpdateColumn("version", v.Version)
	}

	return nil
}

// AfterCreate sets the internal read version to reflect the created version
func (v *Versioned) AfterCreate(tx *gorm.DB) error {
	if tx.Error != nil {
		return nil
	}

	v.readVersion = v.Version

	return nil
}

// AfterFind sets the internal read version based on the retrieved version
func (v *Versioned) AfterFind(tx *gorm.DB) error {
	if tx.Error != nil {
		return nil
	}

	v.readVersion = v.Version

	return nil
}

func (v *Versioned) assertLockValidity(tx *gorm.DB, updateVersion bool) error {
	tx.Statement.Where("version = ?", v.readVersion)

	if updateVersion {
		v.Version = v.readVersion + 1
		tx.Statement.AddClause(clause.Set{{Column: clause.Column{Name: "version"}, Value: v.Version}})
	}

	return nil
}

func (v *Versioned) ensureRowsAffected(tx *gorm.DB) error {
	if tx.Statement.DB.RowsAffected < 1 {
		return ErrConcurrentModification
	}

	return nil
}
