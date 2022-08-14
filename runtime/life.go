package runtime

var (
	cleanups map[string]func()
	sequence []string
)

// OnClose registers a function to be called on butler-close/shutdown
func OnClose(name string, h func()) {
	if cleanups == nil {
		cleanups = make(map[string]func())
	}
	if _, ok := cleanups[name]; !ok {
		sequence = append([]string{name}, sequence...)
		cleanups[name] = h
	}
}

// Close calls all registered handlers from OnClose
func Close() {
	for _, name := range sequence {
		//log.Debug().Msgf("-- cleanup %s", name)
		cleanups[name]()
	}
}
