package index

import (
	"fmt"
	"strconv"

	"github.com/bwireman/archivist/internal/codemap"
	"github.com/bwireman/archivist/internal/store"
)

// ReindexPrompt is set when the stored code map was built by an older
// extractor than this binary. A newer stored map is left alone. An empty
// checkout with no version stays quiet.
func ReindexPrompt(st *store.Store) (string, error) {
	if st == nil {
		return "", nil
	}
	raw, ok, err := st.GetMeta(store.MetaCodemapVersion)
	if err != nil {
		return "", err
	}
	if !ok {
		n, err := st.FileCount()
		if err != nil || n == 0 {
			return "", err
		}
		return "Code map has no extractor version. Run `archivist index` to rebuild it.", nil
	}
	have, err := strconv.Atoi(raw)
	if err != nil || have < codemap.Version {
		return fmt.Sprintf("Code map extractor is %s; this binary is %d. Run `archivist index` to rebuild it.", raw, codemap.Version), nil
	}
	return "", nil
}
