package closer

func CloseAll() {
	globalCloser.CloseAll()
}
