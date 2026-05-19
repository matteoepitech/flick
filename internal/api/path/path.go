/*
** FLICK PROJECT, 2026
** flick/internal/api/path/path
** File description:
** Shared filesystem paths for the API
 */

package path

import "os"

// Unexported package-level paths. They are initialized exactly once in init().
var (
	homeDir   string
	flickDir  string
	dataDir   string
	cacheFile string
)

// init: Resolve the filesystem paths used by the API at package load time.
// Panics if the home directory cannot be resolved, since the API cannot run without it.
func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("path: cannot resolve home directory: " + err.Error())
	}
	homeDir = home
	flickDir = home + "/.flick/"
	dataDir = flickDir + "data/"
	cacheFile = flickDir + "server-codes.cache"
}

// HomeDir: Get the home dir path.
//
// Returns:
// - result1 (string): The internal home dir variable.
func GetHomeDir() string {
	return homeDir
}

// GetFlickDir: Get the flick dir path.
//
// Returns:
// - result1 (string): The internal flick dir variable.
func GetFlickDir() string {
	return flickDir
}

// GetDataDir: Get the data dir path.
//
// Returns:
// - result1 (string): The internal data dir variable.
func GetDataDir() string {
	return dataDir
}

// GetCacheFile: Get the cache file path.
//
// Returns:
// - result1 (string): The internal cache file variable.
func GetCacheFile() string {
	return cacheFile
}
