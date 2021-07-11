package tests

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/omaskery/optimistic-gorm/optimistic"
)

var (
	TestID        uint = 1
	NonExistantID      = 999999
)

type TestModel struct {
	gorm.Model
	optimistic.Versioned

	Value int
}

var _ = Describe("Tests", func() {
	var tempDir string
	var db *gorm.DB

	JustBeforeEach(func() {
		log.SetOutput(GinkgoWriter)

		var err error

		tempDir, err = ioutil.TempDir("", "tests-")
		Expect(err).To(Succeed())
		log.Printf("created temporary directory file://%s", tempDir)

		dbPath := path.Join(tempDir, "test.sqlite3")
		log.Printf("creating database at file://%s", dbPath)
		dialector := sqlite.Open(dbPath)
		db, err = gorm.Open(dialector, &gorm.Config{
			Logger: logger.New(log.New(GinkgoWriter, "\r\n", log.LstdFlags), logger.Config{
				LogLevel: logger.Info,
				Colorful: true,
			}),
			SkipDefaultTransaction: true,
		})
		Expect(err).To(Succeed())
		db = db.Debug()

		sqliteDB, err := db.DB()
		Expect(err).To(Succeed())
		sqliteDB.SetMaxOpenConns(1)

		Expect(db.AutoMigrate(&TestModel{})).To(Succeed())
	})

	JustAfterEach(func() {
		log.Printf("closing database")
		sqliteDB, err := db.DB()
		Expect(err).To(Succeed())
		Expect(sqliteDB.Close()).To(Succeed())

		log.Printf("removing temporary directory file://%s", tempDir)
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	It("attempting to find a non-existent model doesn't break", func() {
		m := TestModel{}
		Expect(db.Transaction(func(tx *gorm.DB) error {
			Expect(tx.Where("id = ?", NonExistantID).Find(&m).Error).To(Succeed())
			Expect(tx.RowsAffected).To(BeNumerically("==", 0))
			return nil
		})).To(Succeed())
	})

	When("an entry is created", func() {
		var m *TestModel

		JustBeforeEach(func() {
			m = &TestModel{
				Model: gorm.Model{
					ID: TestID,
				},
				Value: 100,
			}
			log.Printf("creating database entry")
			Expect(db.Transaction(func(tx *gorm.DB) error {
				Expect(tx.Create(m).Error).To(Succeed())
				return nil
			})).To(Succeed())

			log.Printf("created database entry: %+v", m)
		})

		It("has the expected initial version in memory", func() {
			Expect(m.Version).To(BeNumerically("==", 1))
		})

		It("has the expected initial version in the database", func() {
			Expect(db.Transaction(func(tx *gorm.DB) error {
				Expect(tx.First(&m).Error).To(Succeed())
				return nil
			})).To(Succeed())

			Expect(m.Version).To(BeNumerically("==", 1))
		})

		When("the entry is updated", func() {
			JustBeforeEach(func() {
				m.Value = 1000
				log.Printf("updating database entry to: %+v", m)
				Expect(db.Transaction(func(tx *gorm.DB) error {
					Expect(tx.Updates(m).Error).To(Succeed())
					return nil
				})).To(Succeed())
			})

			It("it has a higher version number in memory", func() {
				log.Printf("in-memory updated model is now: %+v", m)
				Expect(m.Version).To(BeNumerically("==", 2))
			})

			It("it has persisted a higher version number", func() {
				Expect(db.Transaction(func(tx *gorm.DB) error {
					Expect(tx.First(&m).Error).To(Succeed())
					return nil
				})).To(Succeed())

				log.Printf("persisted updated model is now: %+v", m)
				Expect(m.Version).To(BeNumerically("==", 2))
			})
		})

		When("the entry is (soft) deleted", func() {
			JustBeforeEach(func() {
				log.Printf("deleting database entry: %+v", m)
				Expect(db.Transaction(func(tx *gorm.DB) error {
					Expect(tx.Delete(m).Error).To(Succeed())
					return nil
				})).To(Succeed())
			})

			It("it has a higher version number in memory", func() {
				log.Printf("in-memory updated model is now: %+v", m)
				Expect(m.Version).To(BeNumerically("==", 2))
			})

			It("it has persisted a higher version number", func() {
				Expect(db.Transaction(func(tx *gorm.DB) error {
					Expect(tx.Unscoped().First(&m).Error).To(Succeed())
					return nil
				})).To(Succeed())

				log.Printf("persisted updated model is now: %+v", m)
				Expect(m.Version).To(BeNumerically("==", 2))
			})
		})

		It("does not interfere with 'hard' deletion", func() {
			Expect(db.Transaction(func(tx *gorm.DB) error {
				Expect(tx.Unscoped().Delete(m).Error).To(Succeed())
				return nil
			})).To(Succeed())
		})

		When("there are concurrent modifications", func() {
			updating := func(model *TestModel, tx *gorm.DB) error {
				model.Value += 100
				return tx.Updates(model).Error
			}
			softDeletion := func(model *TestModel, tx *gorm.DB) error {
				return tx.Delete(model).Error
			}
			hardDeletion := func(model *TestModel, tx *gorm.DB) error {
				return tx.Unscoped().Delete(model).Error
			}

			modifications := map[string]func(model *TestModel, tx *gorm.DB) error{
				"updating":      updating,
				"soft-deletion": softDeletion,
				"hard-deletion": hardDeletion,
			}

			for aName, aModification := range modifications {
				for bName, bModification := range modifications {
					It(fmt.Sprintf("detects [%s vs %s]", aName, bName), func() {
						a := &TestModel{}
						b := &TestModel{}

						// read the model at the same initial version
						Expect(db.Transaction(func(tx *gorm.DB) error {
							Expect(tx.Where("id = ?", TestID).First(a).Error).To(Succeed())
							Expect(tx.Where("id = ?", TestID).First(b).Error).To(Succeed())
							return nil
						})).To(Succeed())

						Expect(db.Transaction(func(tx *gorm.DB) error {
							return aModification(a, tx)
						})).To(Succeed())

						Expect(db.Transaction(func(tx *gorm.DB) error {
							return bModification(b, tx)
						})).To(MatchError(optimistic.ErrConcurrentModification))
					})
				}
			}
		})
	})
})
