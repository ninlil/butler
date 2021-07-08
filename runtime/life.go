package runtime

import "github.com/ninlil/butler/log"

var (
	cleanups map[string]func()
)

// OnClose registers a function to be called on butler-close/shutdown
func OnClose(name string, h func()) {
	if cleanups == nil {
		cleanups = make(map[string]func())
	}
	cleanups[name] = h
}

// Close calls all registered handlers from OnClose
func Close() {
	for name, cleanup := range cleanups {
		log.Debug().Msgf("-- cleanup %s", name)
		cleanup()
	}
}
