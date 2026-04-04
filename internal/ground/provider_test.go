package ground

import (
	"github.com/MikkoParkkola/trvl/internal/models"
)

// Compile-time assertion: DefaultProvider implements models.GroundSearcher.
var _ models.GroundSearcher = (*DefaultProvider)(nil)
