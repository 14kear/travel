package flights

import (
	"github.com/MikkoParkkola/trvl/internal/models"
)

// Compile-time assertion: DefaultProvider implements models.FlightSearcher.
var _ models.FlightSearcher = (*DefaultProvider)(nil)
