package ground

var marketedProviderNames = []string{
	"flixbus",
	"regiojet",
	"eurostar",
	"db",
	"oebb",
	"ns",
	"vr",
	"sncf",
	"trainline",
	"transitous",
	"renfe",
	"tallink",
	"vikingline",
	"eckeroline",
	"stenaline",
	"dfds",
}

// MarketedProviderNames returns the user-facing ground-provider catalog used by
// public docs and claims. The returned slice is a copy so callers cannot mutate
// the package-level source of truth.
func MarketedProviderNames() []string {
	return append([]string(nil), marketedProviderNames...)
}

// MarketedProviderCount returns the user-facing ground-provider count.
func MarketedProviderCount() int {
	return len(marketedProviderNames)
}
