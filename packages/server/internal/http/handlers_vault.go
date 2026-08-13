package httpapi

import (
	"errors"
	"net/http"

	appvault "github.com/webcloster-dev/ai-reviewer/internal/app/vault"
)

// vaultStatusResp reports whether the secret vault is initialized and whether it
// is protected by a password (vs a key file beside the database).
type vaultStatusResp struct {
	Initialized       bool `json:"initialized"`
	PasswordProtected bool `json:"passwordProtected"`
}

// vaultStatus returns the current vault mode. It never returns any key material.
func (s *Server) vaultStatus(w http.ResponseWriter, r *http.Request) {
	if s.vault == nil {
		writeErr(w, errors.New("vault management is not available"), http.StatusNotImplemented)
		return
	}
	st, err := s.vault.Status(r.Context())
	if err != nil {
		writeErr(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, vaultStatusResp{Initialized: st.Initialized, PasswordProtected: st.PasswordProtected})
}

// changeVaultSecretResp is returned after a successful key change. `warning` is
// set when the operator must act to keep the next restart working.
type changeVaultSecretResp struct {
	PasswordProtected bool   `json:"passwordProtected"`
	Warning           string `json:"warning,omitempty"`
}

// changeVaultSecret rotates the master key. The current password must be correct
// (ignored in key-file mode); an empty newPassword switches to key-file mode.
// Every secret is re-encrypted under the new key in one transaction.
func (s *Server) changeVaultSecret(w http.ResponseWriter, r *http.Request) {
	if s.vault == nil {
		writeErr(w, errors.New("vault management is not available"), http.StatusNotImplemented)
		return
	}
	var in struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err, http.StatusBadRequest)
		return
	}
	st, err := s.vault.ChangeSecret(r.Context(), in.OldPassword, in.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, appvault.ErrWrongPassword):
			writeErr(w, err, http.StatusUnauthorized)
		case errors.Is(err, appvault.ErrNotInitialized):
			writeErr(w, err, http.StatusConflict)
		default:
			// A re-encryption failure leaves the vault unchanged (the swap only
			// happens after the transaction commits), so this is safe to surface.
			writeErr(w, err, http.StatusInternalServerError)
		}
		return
	}
	resp := changeVaultSecretResp{PasswordProtected: st.PasswordProtected}
	if st.PasswordProtected {
		// The boot password is read from AIR_PASSWORD; the stored key just changed,
		// so the env must be updated to the new password or the next start fails to
		// unlock. Key-file mode needs no such change (the key lives beside the DB).
		resp.Warning = "Update AIR_PASSWORD to the new password before restarting, or the vault will not unlock."
	}
	writeJSON(w, http.StatusOK, resp)
}
