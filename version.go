package ui

import (
	_ "embed"
)

//go:generate sh -c "head -n 1 CHANGELOG.md | grep -oP '# \\[\\K[^]]+' > VERSION.txt"
//go:embed VERSION.txt
var Version string
