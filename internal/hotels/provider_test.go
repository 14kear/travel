package hotels

import (
	"github.com/MikkoParkkola/trvl/internal/models"
)

// Compile-time assertion: DefaultProvider implements models.HotelSearcher.
var _ models.HotelSearcher = (*DefaultProvider)(nil)
