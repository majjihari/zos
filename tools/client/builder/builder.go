package builder

// Builder interface
type Builder interface {
	Save(name string) error
	Deploy() error
}
