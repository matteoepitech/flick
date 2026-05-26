/*
** FLICK PROJECT, 2026
** flick/internal/api/serverconfig/serverconfig
** File description:
** Server-side configuration
 */

package serverconfig

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/matteoepitech/flick/internal/api/utils"
)

// Server configuration template
type Configuration struct {
	Persistence             bool   `json:"persistence"`
	MaxFileSizeMb           int    `json:"max_file_size_mb" validate:"required,gte=0"`
	DefaultExpiration       string `json:"default_expiration" validate:"required,duration" user:"true"`
	MaxExpiration           string `json:"max_expiration" validate:"required,duration"`
	AllowMultipleDownloads  bool   `json:"allow_multiple_downloads"`
	DefaultDownloadCount    int    `json:"default_download_count" validate:"required,gte=1" user:"true"`
	MaxDownloadCount        int    `json:"max_download_count" validate:"required,gtefield=DefaultDownloadCount"`
	RequirePassword         bool   `json:"require_password"`
	ActivateRateLimit       bool   `json:"activate_rate_limit"`
	MaxGenerationKeyPerHour int    `json:"max_generation_key_per_hour" validate:"required,gte=0"`
	MaxUploadPerHourPerKey  int    `json:"max_upload_per_hour_per_key" validate:"required,ltefield=MaxUploadPerHourPerIP"`
	MaxUploadPerHourPerIP   int    `json:"max_upload_per_hour_per_ip" validate:"required,gtfield=MaxUploadPerHourPerKey"`
	MaxUploadPerHour        int    `json:"max_upload_per_hour" validate:"required,gtfield=MaxUploadPerHourPerIP"`
}

// Server configuration default values
var Conf Configuration = Configuration{
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
}

// Validate for the struct tag.
var validate = validator.New()

// init: Init function for the serverconfig package.
func init() {
	validate.RegisterValidation("duration", func(fl validator.FieldLevel) bool {
		_, err := utils.ParseExpirationTime(fl.Field().String())
		return err == nil
	})
}

// UserFields: Returns only the configuration fields tagged with `user:"true"`.
//
// Params:
// - c (Configuration): The configuration to filter.
//
// Returns:
// - map[string]any: The user-facing fields keyed by their JSON name.
func UserFields(c Configuration) map[string]any {
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

// Validate: Validates the given configuration against the struct tags.
//
// Params:
// - c (*Configuration): The configuration to validate.
//
// Returns:
// - error: The validation error, if any.
func Validate(c *Configuration) error {
	return validate.Struct(c)
}
