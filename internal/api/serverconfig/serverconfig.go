/*
** FLICK PROJECT, 2026
** flick/internal/api/serverconfig/serverconfig
** File description:
** Server-side configuration
 */

package serverconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/path"
	"github.com/Flick-Corp/flick/internal/api/utils"
	"github.com/go-playground/validator/v10"
)

// Server configuration template
type Configuration struct {
	Persistence             bool   `json:"persistence"`
	MaxFileSizeMb           int    `json:"max_file_size_mb" validate:"required,gte=0" user:"true"`
	DefaultExpiration       string `json:"default_expiration" validate:"required,duration" user:"true"`
	MaxExpiration           string `json:"max_expiration" validate:"required,duration" user:"true"`
	AllowMultipleDownloads  bool   `json:"allow_multiple_downloads" user:"true"`
	DefaultDownloadCount    int    `json:"default_download_count" validate:"required,gte=1" user:"true"`
	MaxDownloadCount        int    `json:"max_download_count" validate:"required,gtefield=DefaultDownloadCount" user:"true"`
	RequirePassword         bool   `json:"require_password"`
	ActivateRateLimit       bool   `json:"activate_rate_limit"`
	MaxGenerationKeyPerHour int    `json:"max_generation_key_per_hour" validate:"required,gte=0"`
	MaxUploadPerHourPerKey  int    `json:"max_upload_per_hour_per_key" validate:"required,ltefield=MaxUploadPerHourPerIP"`
	MaxUploadPerHourPerIP   int    `json:"max_upload_per_hour_per_ip" validate:"required,gtfield=MaxUploadPerHourPerKey"`
	MaxUploadPerHour        int    `json:"max_upload_per_hour" validate:"required,gtfield=MaxUploadPerHourPerIP"`
	AnonymousQuotaMb        int    `json:"anonymous_quota_mb" validate:"gte=0" user:"true"`
	UserQuotaMb             int    `json:"user_quota_mb"      validate:"gte=0" user:"true"`
	GroupQuotaMb            int    `json:"group_quota_mb"     validate:"gte=0" user:"true"`
}

// Server configuration default values
var DefaultConfig Configuration = Configuration{
	Persistence:             true,
	MaxFileSizeMb:           1000,
	DefaultExpiration:       "15m",
	MaxExpiration:           "4h",
	AllowMultipleDownloads:  false,
	DefaultDownloadCount:    1,
	MaxDownloadCount:        5,
	RequirePassword:         false,
	ActivateRateLimit:       true,
	MaxGenerationKeyPerHour: 5,
	MaxUploadPerHourPerKey:  5,
	MaxUploadPerHourPerIP:   30,
	MaxUploadPerHour:        100,
	AnonymousQuotaMb:        1000,
	UserQuotaMb:             5000,
	GroupQuotaMb:            10000,
}

// Server configuration currently in use
var Conf Configuration = DefaultConfig

// Validate for the struct tag.
var validate = validator.New()

// init: Init function for the serverconfig package.
func init() {
	validate.RegisterValidation("duration", func(fl validator.FieldLevel) bool {
		_, err := utils.ParseExpirationTime(fl.Field().String())
		return err == nil
	})
}

// WriteDefaultConfig: Writes the default server configuration.
func WriteDefaultConfig() {
	dir := path.GetFlickDir()
	if _, err := os.Stat(filepath.Join(dir, "server-config.json")); err == nil {
		logging.LogInfo("Server configuration file already exists")
		return
	}
	data, _ := json.MarshalIndent(DefaultConfig, "", " ")
	os.WriteFile(filepath.Join(dir, "server-config.json"), data, 0644)
}

// LoadServerConfigFromDisk: Loads the server configuration file into Conf.
//
// Returns:
// - error: The loading error, if any.
func LoadServerConfigFromDisk() error {
	dir := path.GetFlickDir()
	data, err := os.ReadFile(filepath.Join(dir, "server-config.json"))
	if err != nil {
		return logging.LogInfoError("Cannot read server configuration: %v", err)
	}

	var conf Configuration
	if err := json.Unmarshal(data, &conf); err != nil {
		return logging.LogInfoError("Cannot parse server configuration: %v", err)
	}
	Conf = conf

	return nil
}

// FilterUserFields: Returns only the configuration fields tagged with user:"true".
//
// Params:
// - c (Configuration): The configuration to filter.
//
// Returns:
// - map[string]any: The user-facing fields keyed by their JSON name.
func FilterUserFields(c Configuration) map[string]any {
	out := make(map[string]any)
	t := reflect.TypeFor[Configuration]()
	v := reflect.ValueOf(c)

	for i := range t.NumField() {
		field := t.Field(i)
		if field.Tag.Get("user") != "true" {
			continue
		}
		jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonName == "" || jsonName == "-" {
			jsonName = field.Name
		}
		out[jsonName] = v.Field(i).Interface()
	}
	return out
}

// Validate: Validates the configuration against the struct tags.
//
// Returns:
// - error: The validation error, if any.
func (c *Configuration) Validate() error {
	return validate.Struct(c)
}
