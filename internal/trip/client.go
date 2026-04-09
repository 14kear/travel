package trip

import "github.com/MikkoParkkola/trvl/internal/batchexec"

// Compound trip planners fan out through multiple large batchexecute responses.
// Reusing one fresh client preserves connection reuse and shared rate limiting
// within the command, while disabling the raw-body cache avoids retaining
// payloads that are not reused across these one-shot searches.
func newCompoundSearchClient() *batchexec.Client {
	client := batchexec.NewClient()
	client.SetNoCache(true)
	return client
}
