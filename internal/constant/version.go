package constant

import (
	"fmt"

	"github.com/ghinknet/toolbox/crypto/fingerprint"
)

const Name = "GoPE"

var Version = [3]int{1, 0, 0}
var VersionText = fmt.Sprintf("%d.%d.%d%s", Version[0], Version[1], Version[2],
	func() string {
		sha256, err := fingerprint.GetExecutableSHA256()
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("-%s", string([]rune(sha256)[:10]))
	}(),
)
