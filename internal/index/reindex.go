package index

import (
	"fmt"
	"strconv"

	"github.com/bwireman/archivist/internal/codemap"
	"github.com/bwireman/archivist/internal/store"
)

// ReindexPrompt is set when the stored code map was built by an older
// extractor than this binary. checkout is the canonical checkout path;
// an empty checkout reads the legacy global meta row. A newer stored map
// is left alone. A checkout with no files and no version stays quiet.
func ReindexPrompt(st *store.Store, checkout string) (string, error) {
	if st == nil {
		return "", nil
	}
	raw, ok, err := codemapVersion(st, checkout)
	if err != nil {
		return "", err
	}
	if !ok {
		n, err := codemapFileCount(st, checkout)
		if err != nil || n == 0 {
			return "", err
		}
		return "Code map has no extractor version. Run `archivist index` or call MCP index to rebuild it.", nil
	}
	have, err := strconv.Atoi(raw)
	if err != nil || have < codemap.Version {
		return fmt.Sprintf("Code map extractor is %s; this binary is %d. Run `archivist index` or call MCP index to rebuild it.", raw, codemap.Version), nil
	}
	return "", nil
}

func codemapVersion(st *store.Store, checkout string) (string, bool, error) {
	if checkout == "" {
		return st.GetMeta(store.MetaCodemapVersion)
	}
	return st.GetCheckoutMeta(checkout, store.MetaCodemapVersion)
}

func codemapFileCount(st *store.Store, checkout string) (int, error) {
	if checkout == "" {
		return st.FileCount()
	}
	return st.CheckoutFileCount(checkout)
}
