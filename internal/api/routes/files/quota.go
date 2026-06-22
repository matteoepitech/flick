/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/quota
** File description:
** Quota usage route handler
 */

package files

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/path"
	"github.com/matteoepitech/flick/internal/api/quota"
	"github.com/matteoepitech/flick/internal/api/routes"
	"github.com/matteoepitech/flick/internal/api/routes/account"
	"github.com/matteoepitech/flick/internal/api/serverconfig"
)

// QuotaHandler: Build the quota usage handler. With a `group_id` query parameter
// it reports the group's usage (caller must be a maintainer); otherwise it
// reports the usage of the uploader given by the X-Flick-User-ID header. The
// response feeds a "used / limit" indicator in the UI. A limit of 0 means the
// scope is unlimited.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - http.HandlerFunc: The handler function.
func QuotaHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var usedBytes int64
		var limitMb int

		if groupParam := r.URL.Query().Get("group_id"); groupParam != "" {
			var groupID pgtype.UUID
			if err := groupID.Scan(groupParam); err != nil {
				routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
				return
			}
			if _, status, err := account.RequireGroupMaintainer(r.Context(), queries, account.TokenFromHeader(r), groupID); err != nil {
				routes.WriteError(w, status, err.Error())
				return
			}
			u, err := quota.UsedByGroupID(path.GetDataDir(), groupParam)
			if err != nil {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot read quota")
				return
			}
			usedBytes = u
			limitMb = serverconfig.Conf.GroupQuotaMb
		} else {
			rawID, isAnonymous, _, err := resolveUploaderID(r, queries)
			if err != nil {
				routes.WriteError(w, http.StatusBadRequest, "Invalid or unknown user")
				return
			}
			u, err := quota.UsedByUploaderID(path.GetDataDir(), rawID)
			if err != nil {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot read quota")
				return
			}
			usedBytes = u
			limitMb = serverconfig.Conf.UserQuotaMb
			if isAnonymous {
				limitMb = serverconfig.Conf.AnonymousQuotaMb
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"usedBytes": usedBytes,
			"limitMb":   limitMb,
		})
	}
}
