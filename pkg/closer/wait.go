package closer

func Wait() {
	globalCloser.Wait()
}
