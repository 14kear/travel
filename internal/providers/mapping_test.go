package providers

import (
	"testing"
)

// --- stripUnresolvedPlaceholders ---

func TestStripUnresolvedPlaceholders_NoPlaceholders(t *testing.T) {
	in := "https://example.com/search?city=Paris&guests=2"
	got := stripUnresolvedPlaceholders(in)
	if got != in {
		t.Errorf("expected unchanged URL, got %q", got)
	}
}

func TestStripUnresolvedPlaceholders_AmpersandParam(t *testing.T) {
	in := "https://example.com/search?city=Paris&nflt=${nflt}&guests=2"
	want := "https://example.com/search?city=Paris&guests=2"
	got := stripUnresolvedPlaceholders(in)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestStripUnresolvedPlaceholders_QuestionMarkParam(t *testing.T) {
	in := "https://example.com/search?${token}&guests=2"
	// leading ?token= is stripped leaving nothing before &guests
	got := stripUnresolvedPlaceholders(in)
	// Should not contain the placeholder
	if mappingContains(got, "${token}") {
		t.Errorf("expected placeholder removed, got %q", got)
	}
}

func TestStripUnresolvedPlaceholders_MultiplePlaceholders(t *testing.T) {
	in := "https://example.com/?a=${x}&b=${y}&c=real"
	got := stripUnresolvedPlaceholders(in)
	if mappingContains(got, "${x}") || mappingContains(got, "${y}") {
		t.Errorf("expected all placeholders removed, got %q", got)
	}
	if !mappingContains(got, "c=real") {
		t.Errorf("expected real param retained, got %q", got)
	}
}

func TestStripUnresolvedPlaceholders_UnclosedBrace(t *testing.T) {
	// No closing } — should leave the string alone.
	in := "https://example.com/?a=${unclosed"
	got := stripUnresolvedPlaceholders(in)
	if got != in {
		t.Errorf("expected unchanged for unclosed brace, got %q", got)
	}
}

// --- resolveCityID ---

func TestResolveCityID_EmptyLookup(t *testing.T) {
	got := resolveCityID(nil, "Paris")
	if got != "" {
		t.Errorf("expected empty for nil lookup, got %q", got)
	}
}

func TestResolveCityID_EmptyLocation(t *testing.T) {
	m := map[string]string{"paris": "PAR"}
	got := resolveCityID(m, "")
	if got != "" {
		t.Errorf("expected empty for empty location, got %q", got)
	}
}

func TestResolveCityID_ExactMatch(t *testing.T) {
	m := map[string]string{"paris": "PAR"}
	got := resolveCityID(m, "Paris")
	if got != "PAR" {
		t.Errorf("expected PAR, got %q", got)
	}
}

func TestResolveCityID_CaseInsensitive(t *testing.T) {
	m := map[string]string{"berlin": "BER"}
	got := resolveCityID(m, "BERLIN")
	if got != "BER" {
		t.Errorf("expected BER, got %q", got)
	}
}

func TestResolveCityID_PartialContainsKey(t *testing.T) {
	// location "Prague" contains key "prag"
	m := map[string]string{"prag": "PRG"}
	got := resolveCityID(m, "Prague")
	if got != "PRG" {
		t.Errorf("expected PRG for partial match, got %q", got)
	}
}

func TestResolveCityID_KeyContainsLocation(t *testing.T) {
	// key "barcelona city" contains location "barcelona"
	m := map[string]string{"barcelona city": "BCN"}
	got := resolveCityID(m, "Barcelona")
	if got != "BCN" {
		t.Errorf("expected BCN for key-contains-location match, got %q", got)
	}
}

func TestResolveCityID_NoMatch(t *testing.T) {
	m := map[string]string{"rome": "ROM"}
	got := resolveCityID(m, "Tokyo")
	if got != "" {
		t.Errorf("expected empty for no match, got %q", got)
	}
}

func TestResolveCityID_WhitespaceLocation(t *testing.T) {
	m := map[string]string{"paris": "PAR"}
	got := resolveCityID(m, "  paris  ")
	if got != "PAR" {
		t.Errorf("expected PAR after trimming, got %q", got)
	}
}

// --- resolvePropertyType ---

func TestResolvePropertyType_EmptyLookup(t *testing.T) {
	got := resolvePropertyType(nil, "hotel")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestResolvePropertyType_ExactMatch(t *testing.T) {
	m := map[string]string{"hotel": "201"}
	got := resolvePropertyType(m, "hotel")
	if got != "201" {
		t.Errorf("expected 201, got %q", got)
	}
}

func TestResolvePropertyType_CaseInsensitive(t *testing.T) {
	m := map[string]string{"Hostel": "204"}
	got := resolvePropertyType(m, "hostel")
	if got != "204" {
		t.Errorf("expected 204, got %q", got)
	}
}

func TestResolvePropertyType_EmptyType(t *testing.T) {
	m := map[string]string{"hotel": "201"}
	got := resolvePropertyType(m, "")
	if got != "" {
		t.Errorf("expected empty for empty type, got %q", got)
	}
}

// --- stripNonNumeric ---

func TestStripNonNumeric_CurrencySymbol(t *testing.T) {
	got := stripNonNumeric("€ 61")
	if got != "61" {
		t.Errorf("expected 61, got %q", got)
	}
}

func TestStripNonNumeric_AlreadyNumeric(t *testing.T) {
	got := stripNonNumeric("123.45")
	if got != "123.45" {
		t.Errorf("expected 123.45, got %q", got)
	}
}

func TestStripNonNumeric_Negative(t *testing.T) {
	got := stripNonNumeric("-99.5")
	if got != "-99.5" {
		t.Errorf("expected -99.5, got %q", got)
	}
}

func TestStripNonNumeric_AllLetters(t *testing.T) {
	got := stripNonNumeric("EUR")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestStripNonNumeric_Empty(t *testing.T) {
	got := stripNonNumeric("")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// --- bookingBedType ---

func TestBookingBedType(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{1, "single bed"},
		{2, "double bed"},
		{3, "bunk bed"},
		{4, "futon"},
		{5, "sofa bed"},
		{6, "king bed"},
		{7, "queen bed"},
		{0, "bed"},
		{99, "bed"},
		{-1, "bed"},
	}
	for _, tt := range tests {
		got := bookingBedType(tt.code)
		if got != tt.want {
			t.Errorf("bookingBedType(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

// --- extractNeighborhood ---

func TestExtractNeighborhood_Primary(t *testing.T) {
	raw := map[string]any{
		"basicPropertyData": map[string]any{
			"location": map[string]any{
				"neighbourhood": map[string]any{
					"name": "Montmartre",
				},
			},
		},
	}
	got := extractNeighborhood(raw)
	if got != "Montmartre" {
		t.Errorf("expected Montmartre, got %q", got)
	}
}

func TestExtractNeighborhood_Fallback(t *testing.T) {
	raw := map[string]any{
		"basicPropertyData": map[string]any{
			"neighbourhood": map[string]any{
				"name": "Kreuzberg",
			},
		},
	}
	got := extractNeighborhood(raw)
	if got != "Kreuzberg" {
		t.Errorf("expected Kreuzberg, got %q", got)
	}
}

func TestExtractNeighborhood_Missing(t *testing.T) {
	raw := map[string]any{"other": "data"}
	got := extractNeighborhood(raw)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// --- extractBlocksPriceSpread ---

func TestExtractBlocksPriceSpread_NoBlocks(t *testing.T) {
	maxPrice, roomCount := extractBlocksPriceSpread(map[string]any{})
	if maxPrice != 0 || roomCount != 0 {
		t.Errorf("expected 0,0 for missing blocks, got %v,%v", maxPrice, roomCount)
	}
}

func TestExtractBlocksPriceSpread_WithBlocks(t *testing.T) {
	raw := map[string]any{
		"blocks": []any{
			map[string]any{
				"finalPrice": map[string]any{"amount": 120.0},
				"blockId":    map[string]any{"roomId": "101"},
			},
			map[string]any{
				"finalPrice": map[string]any{"amount": 280.0},
				"blockId":    map[string]any{"roomId": "201"},
			},
			map[string]any{
				"finalPrice": map[string]any{"amount": 200.0},
				"blockId":    map[string]any{"roomId": "101"}, // duplicate roomId
			},
		},
	}
	maxPrice, roomCount := extractBlocksPriceSpread(raw)
	if maxPrice != 280.0 {
		t.Errorf("expected maxPrice=280, got %v", maxPrice)
	}
	if roomCount != 2 {
		t.Errorf("expected roomCount=2 (2 distinct room IDs), got %d", roomCount)
	}
}

// --- extractDescription ---

func TestExtractDescription_PropertyDescription(t *testing.T) {
	raw := map[string]any{"propertyDescription": "A lovely hotel"}
	got := extractDescription(raw)
	if got != "A lovely hotel" {
		t.Errorf("expected 'A lovely hotel', got %q", got)
	}
}

func TestExtractDescription_Tagline(t *testing.T) {
	raw := map[string]any{
		"basicPropertyData": map[string]any{
			"tagline": "Best value in Paris",
		},
	}
	got := extractDescription(raw)
	if got != "Best value in Paris" {
		t.Errorf("expected tagline, got %q", got)
	}
}

func TestExtractDescription_Missing(t *testing.T) {
	raw := map[string]any{"other": "value"}
	got := extractDescription(raw)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// --- toFloat64 ---

func TestToFloat64_Float(t *testing.T) {
	if got := toFloat64(3.14); got != 3.14 {
		t.Errorf("expected 3.14, got %v", got)
	}
}

func TestToFloat64_Int(t *testing.T) {
	if got := toFloat64(42); got != 42.0 {
		t.Errorf("expected 42.0, got %v", got)
	}
}

func TestToFloat64_StringNumeric(t *testing.T) {
	if got := toFloat64("123.45"); got != 123.45 {
		t.Errorf("expected 123.45, got %v", got)
	}
}

func TestToFloat64_StringWithCurrency(t *testing.T) {
	// "€ 61" should yield 61.0
	if got := toFloat64("€ 61"); got != 61.0 {
		t.Errorf("expected 61.0 from '€ 61', got %v", got)
	}
}

func TestToFloat64_CompositeRating(t *testing.T) {
	// "4.84 (25)" — firstNumericToken should return "4.84"
	got := toFloat64("4.84 (25)")
	if got != 4.84 {
		t.Errorf("expected 4.84 from '4.84 (25)', got %v", got)
	}
}

func TestToFloat64_NilType(t *testing.T) {
	if got := toFloat64(nil); got != 0 {
		t.Errorf("expected 0 for nil, got %v", got)
	}
}

// --- mapHotelResult ---

func TestMapHotelResult_BasicFields(t *testing.T) {
	raw := map[string]any{
		"name":    "Grand Hotel",
		"price":   120.5,
		"rating":  4.7,
		"address": "1 Main St",
	}
	fields := map[string]string{
		"name":    "name",
		"price":   "price",
		"rating":  "rating",
		"address": "address",
	}
	h := mapHotelResult(raw, fields)
	if h.Name != "Grand Hotel" {
		t.Errorf("Name: got %q, want 'Grand Hotel'", h.Name)
	}
	if h.Price != 120.5 {
		t.Errorf("Price: got %v, want 120.5", h.Price)
	}
	if h.Rating != 4.7 {
		t.Errorf("Rating: got %v, want 4.7", h.Rating)
	}
	if h.Address != "1 Main St" {
		t.Errorf("Address: got %q, want '1 Main St'", h.Address)
	}
}

func TestMapHotelResult_HotelIDFloat(t *testing.T) {
	raw := map[string]any{
		"id": float64(1042748),
	}
	fields := map[string]string{"hotel_id": "id"}
	h := mapHotelResult(raw, fields)
	if h.HotelID != "1042748" {
		t.Errorf("HotelID: got %q, want '1042748'", h.HotelID)
	}
}

func TestMapHotelResult_CurrencyExtraction(t *testing.T) {
	// When no explicit currency field, extract from price string.
	raw := map[string]any{"price_str": "EUR 204"}
	fields := map[string]string{"price": "price_str"}
	h := mapHotelResult(raw, fields)
	if h.Currency != "EUR" {
		t.Errorf("Currency: got %q, want 'EUR'", h.Currency)
	}
}

// --- extractCurrencyCode (via mapHotelResult or directly) ---

func TestExtractCurrencyCode_SymbolPrefix(t *testing.T) {
	raw := map[string]any{"p": "€175"}
	fields := map[string]string{"price": "p"}
	h := mapHotelResult(raw, fields)
	if h.Currency != "EUR" {
		t.Errorf("expected EUR from €175, got %q", h.Currency)
	}
}

func TestExtractCurrencyCode_ThreeLetterSuffix(t *testing.T) {
	raw := map[string]any{"p": "120 USD"}
	fields := map[string]string{"price": "p"}
	h := mapHotelResult(raw, fields)
	if h.Currency != "USD" {
		t.Errorf("expected USD from '120 USD', got %q", h.Currency)
	}
}

// helper to avoid grep-blocked grep usage
func mappingContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
