package flights

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/MikkoParkkola/trvl/internal/batchexec"
	"github.com/MikkoParkkola/trvl/internal/testutil"
)

// TestFiltersProbe makes real batchexecute calls to Google Flights to verify
// that the departure time, emissions, and alliance filters produce different
// result counts when applied.
//
// Positions tested:
//
//	segment[2]   departure time window [startHour, endHour]
//	segment[13]  emissions filter (1 = less emissions only)
//	outer[1][25] alliance filter (["STAR_ALLIANCE"], ["ONEWORLD"], etc.)
//
// For each filter we send: nil (baseline) and one or more constrained values.
// If the constrained value returns fewer flights than nil, the filter works.
// If any probe returns HTTP 400, the wire format is wrong.
func TestFiltersProbe(t *testing.T) {
	testutil.RequireLiveProbe(t)

	client := batchexec.NewClient()
	client.SetNoCache(true)
	client.SetRateLimit(0.5) // pace to stay under rate limits

	searchDate := time.Now().AddDate(0, 0, 21).Format("2006-01-02")
	t.Logf("Route: HEL -> LHR, date: %s", searchDate)

	baseOpts := SearchOptions{Adults: 1}
	baseOpts.defaults()

	// Sanity check: baseline must return flights.
	{
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		res, err := SearchFlightsWithClient(ctx, client, "HEL", "LHR", searchDate, baseOpts)
		if err != nil {
			t.Fatalf("baseline search failed: %v", err)
		}
		t.Logf("Baseline (SearchFlightsWithClient): %d flights", res.Count)
		if res.Count == 0 {
			t.Fatal("baseline returned 0 flights -- route/date unusable")
		}
	}

	// ---- Departure Time ----
	t.Run("DepartureTime", func(t *testing.T) {
		probes := []filterProbe{
			{"nil", nil},
			{"[6,12]_morning", []any{6, 12}},
			{"[18,24]_evening", []any{18, 24}},
			{"[0,6]_redeye", []any{0, 6}},
			{"[12,18]_afternoon", []any{12, 18}},
		}
		results := runSegmentProbes(t, client, "HEL", "LHR", searchDate, baseOpts, 2, probes)
		analyzeResults(t, "DEPART_TIME", results)
	})

	// ---- Emissions (segment[13]) ----
	// Scalar 1 returns 400. Probe alternative wire formats.
	t.Run("Emissions", func(t *testing.T) {
		probes := []filterProbe{
			{"nil", nil},
			{"1_scalar", 1},
			{"[1]_array", []any{1}},
			{"true", true},
			{"[1,nil]", []any{1, nil}},
			{"[nil,1]", []any{nil, 1}},
		}
		results := runSegmentProbes(t, client, "HEL", "LHR", searchDate, baseOpts, 13, probes)
		analyzeResults(t, "EMISSIONS_seg13", results)
	})

	// ---- Emissions at segment[14] (maybe off-by-one?) ----
	t.Run("Emissions_pos14", func(t *testing.T) {
		probes := []filterProbe{
			{"nil", nil},
			{"1_scalar", 1},
			{"[1]_array", []any{1}},
		}
		// Position 14 is currently hardcoded to 3 in buildSegment.
		// Probe with emissions values there instead.
		results := runSegmentProbes(t, client, "HEL", "LHR", searchDate, baseOpts, 14, probes)
		analyzeResults(t, "EMISSIONS_seg14", results)
	})

	// ---- Alliance (outer[1][25]) ----
	// String arrays return 400. Probe alternative formats.
	t.Run("Alliance", func(t *testing.T) {
		probes := []filterProbe{
			{"nil", nil},
			{"str_STAR_ALLIANCE", []any{"STAR_ALLIANCE"}},
			{"int_1", []any{1}},                           // maybe integer codes?
			{"int_2", []any{2}},                           // ONEWORLD as int?
			{"int_3", []any{3}},                           // SKYTEAM as int?
			{"nested_str", []any{[]any{"STAR_ALLIANCE"}}}, // nested array?
			{"nested_int", []any{[]any{1}}},               // nested int array?
			{"scalar_1", 1},                               // scalar int
			{"scalar_str", "STAR_ALLIANCE"},               // scalar string
		}
		results := runSettingsProbes(t, client, "HEL", "LHR", searchDate, baseOpts, 25, probes)
		analyzeResults(t, "ALLIANCE_pos25", results)
	})
}

// filterProbe defines a single probe: a name and a value to patch in.
type filterProbe struct {
	name  string
	value any
}

// probeResult captures one probe's outcome.
type probeResult struct {
	name     string
	status   int
	count    int
	bodySize int
	err      error
}

// runSegmentProbes patches a position inside segment[0] (the outbound segment,
// located at outer[1][13][0]) and fires the request.
func runSegmentProbes(t *testing.T, client *batchexec.Client, origin, dest, date string, opts SearchOptions, segPos int, probes []filterProbe) []probeResult {
	t.Helper()
	results := make([]probeResult, len(probes))

	for i, p := range probes {
		t.Run(p.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			filters := buildFilters(origin, dest, date, opts)
			patched := patchSegmentPosition(t, filters, segPos, p.value)

			encoded, err := batchexec.EncodeFlightFilters(patched)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}

			status, body, err := client.SearchFlights(ctx, encoded)
			if err != nil {
				t.Logf("FAILED %s: %v", p.name, err)
				results[i] = probeResult{name: p.name, err: err}
				return
			}

			count := 0
			if status == 200 {
				count = countFlightsProbe(t, body)
			} else {
				t.Logf("raw body: %s", truncateBytes(body, 300))
			}

			results[i] = probeResult{
				name:     p.name,
				status:   status,
				count:    count,
				bodySize: len(body),
			}
			t.Logf("%-20s  status=%d  flights=%d  body=%d bytes",
				p.name, status, count, len(body))
		})
	}
	return results
}

// runSettingsProbes patches a position in outer[1] (the settings array) and
// fires the request.
func runSettingsProbes(t *testing.T, client *batchexec.Client, origin, dest, date string, opts SearchOptions, settingsPos int, probes []filterProbe) []probeResult {
	t.Helper()
	results := make([]probeResult, len(probes))

	for i, p := range probes {
		t.Run(p.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			filters := buildFilters(origin, dest, date, opts)
			patched := patchSettingsPosition(t, filters, settingsPos, p.value)

			encoded, err := batchexec.EncodeFlightFilters(patched)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}

			status, body, err := client.SearchFlights(ctx, encoded)
			if err != nil {
				t.Logf("FAILED %s: %v", p.name, err)
				results[i] = probeResult{name: p.name, err: err}
				return
			}

			count := 0
			if status == 200 {
				count = countFlightsProbe(t, body)
			} else {
				t.Logf("raw body: %s", truncateBytes(body, 300))
			}

			results[i] = probeResult{
				name:     p.name,
				status:   status,
				count:    count,
				bodySize: len(body),
			}
			t.Logf("%-20s  status=%d  flights=%d  body=%d bytes",
				p.name, status, count, len(body))
		})
	}
	return results
}

// patchSegmentPosition round-trips filters through JSON, then replaces
// outer[1][13][0][segPos] (the outbound segment at the given position).
func patchSegmentPosition(t *testing.T, filters any, segPos int, val any) any {
	t.Helper()

	data, err := json.Marshal(filters)
	if err != nil {
		t.Fatalf("marshal filters: %v", err)
	}
	var tree []any
	if err := json.Unmarshal(data, &tree); err != nil {
		t.Fatalf("unmarshal filters: %v", err)
	}

	settings, ok := tree[1].([]any)
	if !ok {
		t.Fatalf("outer[1] is not []any: %T", tree[1])
	}

	segments, ok := settings[13].([]any)
	if !ok {
		t.Fatalf("settings[13] (segments) is not []any: %T", settings[13])
	}

	seg, ok := segments[0].([]any)
	if !ok {
		t.Fatalf("segments[0] is not []any: %T", segments[0])
	}

	if len(seg) <= segPos {
		t.Fatalf("segment too short (%d), need index %d", len(seg), segPos)
	}

	seg[segPos] = val
	segments[0] = seg
	settings[13] = segments
	tree[1] = settings
	return tree
}

// patchSettingsPosition round-trips filters through JSON, then replaces
// outer[1][settingsPos].
func patchSettingsPosition(t *testing.T, filters any, settingsPos int, val any) any {
	t.Helper()

	data, err := json.Marshal(filters)
	if err != nil {
		t.Fatalf("marshal filters: %v", err)
	}
	var tree []any
	if err := json.Unmarshal(data, &tree); err != nil {
		t.Fatalf("unmarshal filters: %v", err)
	}

	settings, ok := tree[1].([]any)
	if !ok {
		t.Fatalf("outer[1] is not []any: %T", tree[1])
	}

	if len(settings) <= settingsPos {
		t.Fatalf("settings too short (%d), need index %d", len(settings), settingsPos)
	}

	settings[settingsPos] = val
	tree[1] = settings
	return tree
}

// countFlightsProbe decodes a raw Google Flights response and returns the
// flight count, or 0 on decode error.
func countFlightsProbe(t *testing.T, body []byte) int {
	t.Helper()

	inner, err := batchexec.DecodeFlightResponse(body)
	if err != nil {
		t.Logf("decode response: %v", err)
		return 0
	}

	rawFlights, err := batchexec.ExtractFlightData(inner)
	if err != nil {
		t.Logf("extract flights: %v", err)
		return 0
	}

	return len(rawFlights)
}

// analyzeResults logs a summary table and highlights key comparisons.
func analyzeResults(t *testing.T, label string, results []probeResult) {
	t.Helper()

	t.Log("")
	t.Logf("=== %s RESULTS ===", label)
	t.Logf("%-20s  %6s  %6s  %8s", "FILTER", "STATUS", "COUNT", "BODY")
	for _, r := range results {
		if r.err != nil {
			t.Logf("%-20s  %6s  %6s  %8s  %v", r.name, "ERR", "-", "-", r.err)
		} else {
			t.Logf("%-20s  %6d  %6d  %8d", r.name, r.status, r.count, r.bodySize)
		}
	}

	// Check for 400s -- indicates broken wire format.
	for _, r := range results {
		if r.status == 400 {
			t.Errorf("BUG: %s=%s returns 400 -- wire format rejected by Google", label, r.name)
		}
	}

	// Compare filtered vs baseline (first result is always nil/baseline).
	if len(results) < 2 {
		return
	}
	baseline := results[0]
	if baseline.err != nil || baseline.status != 200 {
		t.Logf("baseline unusable, skipping comparison")
		return
	}

	t.Log("")
	t.Logf("=== %s ANALYSIS ===", label)
	for _, r := range results[1:] {
		if r.err != nil || r.status != 200 {
			continue
		}
		delta := baseline.count - r.count
		pct := 0.0
		if baseline.count > 0 {
			pct = float64(delta) / float64(baseline.count) * 100
		}
		verdict := "SAME"
		if r.count < baseline.count {
			verdict = fmt.Sprintf("FEWER (-%d, -%.0f%%)", delta, pct)
		} else if r.count > baseline.count {
			verdict = fmt.Sprintf("MORE (+%d)", r.count-baseline.count)
		}
		t.Logf("%-20s  vs baseline: %s  (%d vs %d)",
			r.name, verdict, r.count, baseline.count)
	}
}
