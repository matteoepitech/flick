/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/files/auth
** File description:
** Shared uploader identification, used by the quota route and the tus package
 */

package files

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/Flick-Corp/flick/internal/api/database"
)

// resolveUploaderID: Validate the mandatory X-Flick-User-ID header against the
// anonymous_users and users tables, and return the uploader UUID. The uploader is
// required: a missing, malformed or unknown id is an error.
//
// Params:
// - r (*http.Request): The incoming request.
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (string): The validated uploader UUID.
// - result2 (bool): True when the uploader is an anonymous (not logged-in) user.
// - result3 (bool): True when the uploader is a registered but blocked account.
// - result4 (error): An error if the header is missing, invalid or unknown.
func resolveUploaderID(r *http.Request, queries *database.Queries) (string, bool, bool, error) {
	return ResolveUploaderByID(r.Context(), queries, r.Header.Get("X-Flick-User-ID"))
}

// ResolveUploaderByID: The id-based core behind resolveUploaderID, exported so the
// tus package (whose hooks expose headers, not an *http.Request) can resolve an
// uploader from the same single source of truth.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - uploaderID (string): The raw X-Flick-User-ID value to validate.
//
// Returns:
// - result1 (string): The validated uploader UUID.
// - result2 (bool): True when the uploader is an anonymous (not logged-in) user.
// - result3 (bool): True when the uploader is a registered but blocked account.
// - result4 (error): An error if the id is missing, invalid or unknown.
func ResolveUploaderByID(ctx context.Context, queries *database.Queries, uploaderID string) (string, bool, bool, error) {
	if uploaderID == "" {
		return "", false, false, fmt.Errorf("missing uploader id")
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(uploaderID); err != nil {
		return "", false, false, fmt.Errorf("invalid user id %q: %w", uploaderID, err)
	}

	if _, err := queries.GetAnonymousUserByID(ctx, userUUID); err == nil {
		return uploaderID, true, false, nil
	}

	if user, err := queries.GetUserByID(ctx, userUUID); err == nil {
		return uploaderID, false, user.Blocked, nil
	}

	return "", false, false, fmt.Errorf("unknown user id %q", uploaderID)
}
