# Optimistic GORM [![Go Reference][docs-badge]][docs] [![Build & Test Status][workflows-badge]][workflows]

This little library provides a basic building block for doing optimistic locking when using [GORM][gorm].

# Quickstart

Embed the `optimistic.Versioned` struct into your model struct, and you're done!

```go
type Person struct {
    gorm.Model
    optimistic.Versioned // <-- the magic!

    Name string
    Age  int
}
```

# How it works

## Gist

This library applies optimistic locking by adding a version number to your models.

* Your model starts at version 1 when created, and every subsequent change to that model increments its version number
  by 1.
* As a result, you can tell if a model has changed "since you read it into memory" by seeing if its version has changed
  since you read it.
* You can built this check into update/delete operations by having them only apply if the version number of the row
  matches the version you expect it to be. If it doesn't match, the operation will affect fewer rows than you expect,
  which you can detect and assume (in most cases) to be a concurrent modification error.

## GORM Details

This library works using GORM hooks on the embedded `optimistic.Versioned` struct. When embedded into your model it
means:

1. Created instances of your model will have a default `Version` value of 1
2. Using `BeforeUpdate`/`BeforeDelete` GORM hooks: updates/deletions automatically:
    * Gain a `SET` clause, updating the `Version` previous version + 1.
    * Gain a `WHERE` clause, checking that the row in the database being modified is still the version we originally
      read into memory.
3. Using `AfterUpdate`/`AfterDelete` GORM hooks: updates/deletions check whether the number of rows affected is 0. If
   so, a `optimistic.ErrConcurrentModification` error is returned.

[gorm]: https://gorm.io
[docs]: https://pkg.go.dev/github.com/omaskery/optimistic-gorm
[docs-badge]: https://pkg.go.dev/badge/github.com/omaskery/optimistic-gorm.svg
[workflows]: https://github.com/omaskery/optimistic-gorm/actions/workflows/go.yml
[workflows-badge]: https://github.com/omaskery/optimistic-gorm/actions/workflows/go.yml/badge.svg
