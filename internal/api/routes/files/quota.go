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

	"github.com/Flick-Corp/flick/internal/api/auth"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/quota"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/Flick-Corp/flick/internal/api/serverconfig"
	"github.com/jackc/pgx/v5/pgtype"
)

// QuotaHandler: Get the quota of a group or an upload. Both by ID.
// If the request has the query parameter group_id (?group_id=...) then it will return the group's quota.
// Otherwise it's the uploader's quota.
//
// Params:
// - queries (*database.Queries): The database queries, used to authorize a member when the code is bound to a group.
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

		if groupID := r.URL.Query().Get("group_id"); groupID != "" {
			var groupPgID pgtype.UUID
			if err := groupPgID.Scan(groupID); err != nil {
				routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
				return
			}

			token := auth.GetTokenFromHTTPRequest(r)
			_, status, err := auth.RequireGroupMaintainer(r.Context(), queries, token, groupPgID)
			if err != nil {
				routes.WriteError(w, status, err.Error())
				return
			}

			u, err := quota.CalculateQuotaByGroupID(groupID)
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

			u, err := quota.CalculateQuotaByUploaderID(rawID)
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

		data, err := json.Marshal(map[string]any{
			"usedBytes": usedBytes,
			"limitMb":   limitMb,
		})
		if err != nil {
			logging.LogInfoError("Cannot encode quota response: %v", err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot encode response")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}
