package closer

func Add(f ...func() error) {
	globalCloser.Add(f...)
}
