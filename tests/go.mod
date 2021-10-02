module github.com/omaskery/optimistic-gorm/tests

go 1.16

replace github.com/omaskery/optimistic-gorm => ../

require (
	github.com/omaskery/optimistic-gorm v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	gorm.io/driver/sqlite v1.1.4
	gorm.io/gorm v1.21.15
)
