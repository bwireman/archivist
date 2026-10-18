package shelf

import "example.com/pantry/jar"

// Place sets a jar on the shelf.
func Place(name string) string {
	return jar.Seal(name)
}
