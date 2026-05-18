package meta

import (
	"fmt"
	"log"

	"go.gh.ink/toolbox/crypto/fingerprint"
)

// Name is the CLI name.
const Name = "GoPE"

// Version is the semantic version tuple.
var Version = [3]int{0, 0, 1}

// VersionText includes the version and a short executable hash.
var VersionText = fmt.Sprintf("%d.%d.%d%s", Version[0], Version[1], Version[2],
	func() string {
		sha256, err := fingerprint.GetExecutableSHA256()
		if err != nil {
			log.Fatal(err)
		}
		return fmt.Sprintf("-%s", string([]rune(sha256)[:10]))
	}(),
)
