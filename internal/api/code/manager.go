/*
** FLICK PROJECT, 2026
** flick/internal/api/code/manager
** File description:
** Code manager source file
 */

package code

import (
	"os"
	"time"

	"github.com/matteoepitech/flick/internal/api/metadata"
	"github.com/matteoepitech/flick/internal/api/path"
	"github.com/patrickmn/go-cache"
)

// Cache variable that hold every codes on RAM.
var Cache *cache.Cache

// InitCodeCache: Init the cache by creating the cache, load the current cache on disk and save every expiration.
//
// Returns:
// - result1 (error): Error if loading the cache from disk fails.
func InitCodeCache() error {
	Cache = cache.New(1*time.Hour, 1*time.Minute)
	Cache.OnEvicted(func(key string, value any) {
		_ = SaveCacheManagerFile(path.GetCacheFile())
		metadata.CheckExpirationToRemove(path.GetDataDir())
	})
	return LoadCacheManagerFile(path.GetCacheFile())
}

// AddCodeToList: Add a code in the list of the manager.
//
// Params:
// - code (string): The code to add.
// - exp (string): The expiration string. ("1d", "1h", ...)
func AddCodeToList(code string, exp string) {
	time, err := time.ParseDuration(exp)
	if err != nil {
		return
	}
	Cache.Set(code, struct{}{}, time)
	SaveCacheManagerFile(path.GetCacheFile())
}

// SaveCacheManagerFile: Save the cache from RAM to the disk.
//
// Params:
// - path (string): The path.
//
// Returns:
// - result1 (error): Error if there is.
func SaveCacheManagerFile(path string) error {
	tmp := path + ".tmp"
	if err := Cache.SaveFile(tmp); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadCacheManagerFile: Load the cache from disk to RAM.
//
// Params:
// - path (string): The path.
//
// Returns:
// - result1 (error): Error if there is.
func LoadCacheManagerFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return Cache.LoadFile(path)
	}
	return nil
}

// IsCodeAlreadyExistInList: Does the code exists or not?
//
// Params:
// - code (string): The code to find.
//
// Returns:
// - result1 (bool): true or false if it exists.
func IsCodeAlreadyExistInList(code string) bool {
	_, exist := Cache.Get(code)
	return exist
}
