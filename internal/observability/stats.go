package observability

import (
	"log"
	"sync/atomic"
)

type Stats struct {
	TotalRequests int64
	AuthFailures  int64
	RateLimited   int64
	Streamed      int64
}

var counters = &Stats{}

func IncRequests()    { atomic.AddInt64(&counters.TotalRequests, 1) }
func IncAuthFails()   { atomic.AddInt64(&counters.AuthFailures, 1) }
func IncRateLimited() { atomic.AddInt64(&counters.RateLimited, 1) }
func IncStreamed()    { atomic.AddInt64(&counters.Streamed, 1) }

func Log() {
	log.Printf("requests=%d auth_failures=%d rate_limited=%d streamed=%d",
		atomic.LoadInt64(&counters.TotalRequests),
		atomic.LoadInt64(&counters.AuthFailures),
		atomic.LoadInt64(&counters.RateLimited),
		atomic.LoadInt64(&counters.Streamed),
	)
}
