package gnet

import (
	"bufio"
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"math/rand"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	errorx "github.com/panjf2000/gnet/v2/pkg/errors"
	"github.com/panjf2000/gnet/v2/pkg/logging"
	bbPool "github.com/panjf2000/gnet/v2/pkg/pool/bytebuffer"
	goPool "github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
)

var (
	datagramLen = 1024
	streamLen   = 1024 * 1024
)

type testConf struct {
	et        bool
	etChunk   int
	reuseport bool
	multicore bool
	async     bool
	writev    bool
	clients   int
	lb        LoadBalancing
}

func TestServer(t *testing.T) {
	// start an engine
	// connect 10 clients
	// each client will pipe random data for 1-3 seconds.
	// the writes to the engine will be random sizes. 0KB - 1MB.
	// the engine will echo back the data.
	// waits for graceful connection closing.
	t.Run("poll-LT", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{false, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{false, 0, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{false, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{false, 0, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{false, 0, false, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{false, 0, false, true, true, true, 10, LeastConnections})
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9991"}, &testConf{false, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9992"}, &testConf{false, 0, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9991"}, &testConf{false, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9992"}, &testConf{false, 0, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{false, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{false, 0, false, true, false, false, 10, SourceAddrHash})
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{false, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{false, 0, false, true, true, false, 10, SourceAddrHash})
			})
		})
		t.Run("unix-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{false, 0, false, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{false, 0, false, true, true, true, 10, SourceAddrHash})
			})
		})
	})

	t.Run("poll-ET", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{true, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{true, 0, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{true, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{true, 0, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{true, 0, false, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{true, 0, false, true, true, true, 10, LeastConnections})
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9991"}, &testConf{true, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9992"}, &testConf{true, 0, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9991"}, &testConf{true, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9992"}, &testConf{true, 0, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{true, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{true, 0, false, true, false, false, 10, SourceAddrHash})
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{true, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{true, 0, false, true, true, false, 10, SourceAddrHash})
			})
		})
		t.Run("unix-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{true, 0, false, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{true, 0, false, true, true, true, 10, SourceAddrHash})
			})
		})
	})

	t.Run("poll-ET-chunk", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{true, 1 << 18, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{true, 1 << 19, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{true, 1 << 18, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{true, 1 << 19, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{true, 1 << 18, false, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{true, 1 << 19, false, true, true, true, 10, LeastConnections})
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9991"}, &testConf{true, 1 << 18, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9992"}, &testConf{true, 1 << 19, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9991"}, &testConf{true, 1 << 18, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9992"}, &testConf{true, 1 << 19, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{true, 1 << 18, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{true, 1 << 19, false, true, false, false, 10, SourceAddrHash})
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{true, 1 << 18, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{true, 1 << 19, false, true, true, false, 10, SourceAddrHash})
			})
		})
		t.Run("unix-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{true, 1 << 18, false, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{true, 1 << 19, false, true, true, true, 10, SourceAddrHash})
			})
		})
	})

	t.Run("poll-reuseport-LT", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{false, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{false, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{false, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{false, 0, true, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{false, 0, true, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{false, 0, true, true, true, true, 10, LeastConnections})
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9991"}, &testConf{false, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9992"}, &testConf{false, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9991"}, &testConf{false, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9992"}, &testConf{false, 0, true, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{false, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{false, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{false, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{false, 0, true, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{false, 0, true, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{false, 0, true, true, true, true, 10, LeastConnections})
			})
		})
	})

	t.Run("poll-reuseport-ET", func(t *testing.T) {
		t.Run("tcp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{true, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{true, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{true, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{true, 0, true, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("tcp-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991"}, &testConf{true, 0, true, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9992"}, &testConf{true, 0, true, true, true, true, 10, LeastConnections})
			})
		})
		t.Run("udp", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9991"}, &testConf{true, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9992"}, &testConf{true, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("udp-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9991"}, &testConf{true, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"udp://:9992"}, &testConf{true, 0, true, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{true, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{true, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("unix-async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{true, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{true, 0, true, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("unix-async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet1.sock"}, &testConf{true, 0, true, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"unix://gnet2.sock"}, &testConf{true, 0, true, true, true, true, 10, LeastConnections})
			})
		})
	})

	t.Run("poll-multi-addrs-LT", func(t *testing.T) {
		t.Run("sync", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "udp://:9993", "udp://:9994", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "udp://:9997", "udp://:9998", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("sync-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, false, false, false, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, false, true, false, true, 10, LeastConnections})
			})
		})
		t.Run("async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "udp://:9993", "udp://:9994", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "udp://:9997", "udp://:9998", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, false, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, false, true, true, true, 10, LeastConnections})
			})
		})
	})

	t.Run("poll-multi-addrs-reuseport-LT", func(t *testing.T) {
		t.Run("sync", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "udp://:9993", "udp://:9994", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "udp://:9997", "udp://:9998", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("sync-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, true, false, false, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, true, true, false, true, 10, LeastConnections})
			})
		})
		t.Run("async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "udp://:9993", "udp://:9994", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "udp://:9997", "udp://:9998", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, true, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, true, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{false, 0, true, true, true, true, 10, LeastConnections})
			})
		})
	})

	t.Run("poll-multi-addrs-ET", func(t *testing.T) {
		t.Run("sync", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "udp://:9993", "udp://:9994", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, false, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "udp://:9997", "udp://:9998", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, false, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("sync-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, false, false, false, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, false, true, false, true, 10, LeastConnections})
			})
		})
		t.Run("async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "udp://:9993", "udp://:9994", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, false, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "udp://:9997", "udp://:9998", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, false, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, false, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, false, true, true, true, 10, LeastConnections})
			})
		})
	})

	t.Run("poll-multi-addrs-reuseport-ET", func(t *testing.T) {
		t.Run("sync", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "udp://:9993", "udp://:9994", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, true, false, false, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "udp://:9997", "udp://:9998", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, true, true, false, false, 10, LeastConnections})
			})
		})
		t.Run("sync-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, true, false, false, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, true, true, false, true, 10, LeastConnections})
			})
		})
		t.Run("async", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "udp://:9993", "udp://:9994", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, true, false, true, false, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "udp://:9997", "udp://:9998", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, true, true, true, false, 10, LeastConnections})
			})
		})
		t.Run("async-writev", func(t *testing.T) {
			t.Run("1-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9991", "tcp://:9992", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, true, false, true, true, 10, RoundRobin})
			})
			t.Run("N-loop", func(t *testing.T) {
				runServer(t, []string{"tcp://:9995", "tcp://:9996", "unix://gnet1.sock", "unix://gnet2.sock"}, &testConf{true, 0, true, true, true, true, 10, LeastConnections})
			})
		})
	})
}

type testServer struct {
	*BuiltinEventEngine
	tester       *testing.T
	eng          Engine
	addrs        []string
	multicore    bool
	async        bool
	writev       bool
	nclients     int
	started      int32
	connected    int32
	disconnected int32
	clientActive int32
	workerPool   *goPool.Pool
}

func (s *testServer) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	fd, err := s.eng.Dup()
	if len(s.addrs) > 1 {
		assert.ErrorIsf(s.tester, err, errorx.ErrUnsupportedOp, "dup error")
	} else {
		require.NoErrorf(s.tester, err, "dup error")
		assert.Greaterf(s.tester, fd, 2, "expected fd: > 2, but got: %d", fd)
		assert.NoErrorf(s.tester, SysClose(fd), "close fd error")
	}
	return
}

func (s *testServer) OnOpen(c Conn) (out []byte, action Action) {
	c.SetContext(c)
	atomic.AddInt32(&s.connected, 1)
	out = []byte("sweetness\r\n")
	require.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	require.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
	return
}

func (s *testServer) OnShutdown(_ Engine) {
	fd, err := s.eng.Dup()
	if len(s.addrs) > 1 {
		assert.ErrorIsf(s.tester, err, errorx.ErrUnsupportedOp, "dup error")
	} else {
		require.NoErrorf(s.tester, err, "dup error")
		assert.Greaterf(s.tester, fd, 2, "expected fd: > 2, but got: %d", fd)
		assert.NoErrorf(s.tester, SysClose(fd), "close fd error")
	}
}

func (s *testServer) OnClose(c Conn, err error) (action Action) {
	if err != nil {
		logging.Debugf("error occurred on closed, %v\n", err)
	}

	require.Equal(s.tester, c.Context(), c, "invalid context")

	atomic.AddInt32(&s.disconnected, 1)
	return
}

func (s *testServer) OnTraffic(c Conn) (action Action) {
	if s.async {
		buf := bbPool.Get()
		_, _ = c.WriteTo(buf)
		if c.LocalAddr().Network() == "tcp" || c.LocalAddr().Network() == "unix" {
			// just for test
			_ = c.InboundBuffered()
			_ = c.OutboundBuffered()
			_, _ = c.Discard(1)

			_ = s.workerPool.Submit(
				func() {
					if s.writev {
						mid := buf.Len() / 2
						bs := make([][]byte, 2)
						bs[0] = buf.B[:mid]
						bs[1] = buf.B[mid:]
						_ = c.AsyncWritev(bs, func(c Conn, err error) error {
							if c.RemoteAddr() != nil {
								logging.Debugf("conn=%s done writev: %v", c.RemoteAddr().String(), err)
							}
							bbPool.Put(buf)
							return nil
						})
					} else {
						_ = c.AsyncWrite(buf.Bytes(), func(c Conn, err error) error {
							if c.RemoteAddr() != nil {
								logging.Debugf("conn=%s done write: %v", c.RemoteAddr().String(), err)
							}
							bbPool.Put(buf)
							return nil
						})
					}
				})
			return
		} else if c.LocalAddr().Network() == "udp" {
			_ = s.workerPool.Submit(
				func() {
					_ = c.AsyncWrite(buf.Bytes(), nil)
				})
			return
		}
		return
	}

	buf, _ := c.Next(-1)
	if s.writev {
		mid := len(buf) / 2
		_, _ = c.Writev([][]byte{buf[:mid], buf[mid:]})
	} else {
		_, _ = c.Write(buf)
	}

	// Only for code coverage of testing.
	if !s.multicore {
		assert.NoErrorf(s.tester, c.Flush(), "flush error")
		_ = c.Fd()
		fd, err := c.Dup()
		require.NoErrorf(s.tester, err, "dup error")
		assert.Greaterf(s.tester, fd, 2, "expected fd: > 2, but got: %d", fd)
		assert.NoErrorf(s.tester, SysClose(fd), "close error")
		// TODO(panjf2000): somehow these two system calls will fail with Unix Domain Socket,
		//  returning "invalid argument" error on macOS in Github actions intermittently,
		//  try to figure it out.
		if c.LocalAddr().Network() == "unix" && runtime.GOOS == "darwin" {
			_ = c.SetReadBuffer(streamLen)
			_ = c.SetWriteBuffer(streamLen)
		} else {
			assert.NoErrorf(s.tester, c.SetReadBuffer(streamLen), "set read buffer error")
			assert.NoErrorf(s.tester, c.SetWriteBuffer(streamLen), "set write buffer error")
		}
		if c.LocalAddr().Network() == "tcp" {
			assert.NoErrorf(s.tester, c.SetLinger(1), "set linger error")
			assert.NoErrorf(s.tester, c.SetNoDelay(false), "set no delay error")
			assert.NoErrorf(s.tester, c.SetKeepAlivePeriod(time.Minute), "set keep alive period error")
		}
	}

	return
}

func (s *testServer) OnTick() (delay time.Duration, action Action) {
	delay = 100 * time.Millisecond
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for _, protoAddr := range s.addrs {
			proto, addr, err := parseProtoAddr(protoAddr)
			assert.NoError(s.tester, err)
			for i := 0; i < s.nclients; i++ {
				atomic.AddInt32(&s.clientActive, 1)
				go func() {
					startClient(s.tester, proto, addr, s.multicore, s.async)
					atomic.AddInt32(&s.clientActive, -1)
				}()
			}
		}
	}
	if atomic.LoadInt32(&s.clientActive) == 0 {
		var streamAddrs int
		for _, addr := range s.addrs {
			if !strings.HasPrefix(addr, "udp") {
				streamAddrs++
			}
		}
		streamConns := s.nclients * streamAddrs
		disconnected := atomic.LoadInt32(&s.disconnected)
		if int(disconnected) == streamConns && disconnected == atomic.LoadInt32(&s.connected) {
			action = Shutdown
			s.workerPool.Release()
			require.EqualValues(s.tester, 0, s.eng.CountConnections())
		}
	}
	return
}

func runServer(t *testing.T, addrs []string, conf *testConf) {
	ts := &testServer{
		tester:     t,
		addrs:      addrs,
		multicore:  conf.multicore,
		async:      conf.async,
		writev:     conf.writev,
		nclients:   conf.clients,
		workerPool: goPool.Default(),
	}
	var err error
	if len(addrs) > 1 {
		err = Rotate(ts,
			addrs,
			WithEdgeTriggeredIO(conf.et),
			WithEdgeTriggeredIOChunk(conf.etChunk),
			WithLockOSThread(conf.async),
			WithMulticore(conf.multicore),
			WithReusePort(conf.reuseport),
			WithTicker(true),
			WithTCPKeepAlive(time.Minute),
			WithTCPNoDelay(TCPNoDelay),
			WithLoadBalancing(conf.lb))
	} else {
		err = Run(ts,
			addrs[0],
			WithEdgeTriggeredIO(conf.et),
			WithEdgeTriggeredIOChunk(conf.etChunk),
			WithLockOSThread(conf.async),
			WithMulticore(conf.multicore),
			WithReusePort(conf.reuseport),
			WithTicker(true),
			WithTCPKeepAlive(time.Minute),
			WithTCPNoDelay(TCPDelay),
			WithLoadBalancing(conf.lb))
	}
	assert.NoError(t, err)
}

func startClient(t *testing.T, network, addr string, multicore, async bool) {
	c, err := net.Dial(network, addr)
	require.NoError(t, err)
	defer c.Close()
	rd := bufio.NewReader(c)
	if network != "udp" {
		msg, err := rd.ReadBytes('\n')
		require.NoError(t, err)
		require.Equal(t, string(msg), "sweetness\r\n", "bad header")
	}
	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 2
	logging.Debugf("test duration: %v", duration)
	start := time.Now()
	for time.Since(start) < duration {
		reqData := make([]byte, streamLen)
		if network == "udp" {
			reqData = reqData[:datagramLen]
		}
		_, err = crand.Read(reqData)
		require.NoError(t, err)
		_, err = c.Write(reqData)
		require.NoError(t, err)
		respData := make([]byte, len(reqData))
		_, err = io.ReadFull(rd, respData)
		require.NoError(t, err)
		if !async {
			// require.Equalf(t, reqData, respData, "response mismatch with protocol:%s, multi-core:%t, content of bytes: %d vs %d", network, multicore, string(reqData), string(respData))
			require.Equalf(
				t,
				reqData,
				respData,
				"response mismatch with protocol:%s, multi-core:%t, length of bytes: %d vs %d",
				network,
				multicore,
				len(reqData),
				len(respData),
			)
		}
	}
}

func TestDefaultGnetServer(*testing.T) {
	svr := BuiltinEventEngine{}
	svr.OnBoot(Engine{})
	svr.OnOpen(nil)
	svr.OnClose(nil, nil)
	svr.OnTraffic(nil)
	svr.OnTick()
}

type testBadAddrServer struct {
	*BuiltinEventEngine
}

func (t *testBadAddrServer) OnBoot(_ Engine) (action Action) {
	return Shutdown
}

func TestBadAddresses(t *testing.T) {
	events := new(testBadAddrServer)
	err := Run(events, "tulip://howdy")
	assert.ErrorIs(t, err, errorx.ErrUnsupportedProtocol)
	err = Run(events, "howdy")
	assert.ErrorIs(t, err, errorx.ErrInvalidNetworkAddress)
	err = Run(events, "tcp://")
	assert.ErrorIs(t, err, errorx.ErrInvalidNetworkAddress)
}

func TestTick(t *testing.T) {
	testTick("tcp", ":9989", t)
}

type testTickServer struct {
	*BuiltinEventEngine
	count int
}

func (t *testTickServer) OnTick() (delay time.Duration, action Action) {
	delay = time.Millisecond * 10
	if t.count == 25 {
		action = Shutdown
		return
	}
	t.count++
	return
}

func testTick(network, addr string, t *testing.T) {
	events := &testTickServer{}
	start := time.Now()
	opts := Options{Ticker: true}
	err := Run(events, network+"://"+addr, WithOptions(opts))
	assert.NoError(t, err)
	dur := time.Since(start)
	if dur < 250&time.Millisecond || dur > time.Second {
		t.Logf("bad ticker timing: %d", dur)
	}
}

func TestWakeConn(t *testing.T) {
	testWakeConn(t, "tcp", ":9990")
}

type testWakeConnServer struct {
	*BuiltinEventEngine
	tester  *testing.T
	network string
	addr    string
	conn    chan Conn
	c       Conn
	wake    bool
}

func (t *testWakeConnServer) OnOpen(c Conn) (out []byte, action Action) {
	t.conn <- c
	return
}

func (t *testWakeConnServer) OnClose(Conn, error) (action Action) {
	action = Shutdown
	return
}

func (t *testWakeConnServer) OnTraffic(c Conn) (action Action) {
	_, _ = c.Write([]byte("Waking up."))
	action = -1
	return
}

func (t *testWakeConnServer) OnTick() (delay time.Duration, action Action) {
	if !t.wake {
		t.wake = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			r := make([]byte, 10)
			_, err = conn.Read(r)
			require.NoError(t.tester, err)
		}()
		return
	}
	t.c = <-t.conn
	_ = t.c.Wake(func(c Conn, err error) error {
		logging.Debugf("conn=%s done wake: %v", c.RemoteAddr().String(), err)
		return nil
	})
	delay = time.Millisecond * 100
	return
}

func testWakeConn(t *testing.T, network, addr string) {
	currentLogger, currentFlusher := logging.GetDefaultLogger(), logging.GetDefaultFlusher()
	t.Cleanup(func() {
		logging.SetDefaultLoggerAndFlusher(currentLogger, currentFlusher) // restore
	})

	svr := &testWakeConnServer{tester: t, network: network, addr: addr, conn: make(chan Conn, 1)}
	logger := zap.NewExample()
	err := Run(svr, network+"://"+addr,
		WithTicker(true),
		WithNumEventLoop(2*runtime.NumCPU()),
		WithLogger(logger.Sugar()),
		WithSocketRecvBuffer(4*1024),
		WithSocketSendBuffer(4*1024),
		WithReadBufferCap(2000),
		WithWriteBufferCap(2000))
	assert.NoError(t, err)
	_ = logger.Sync()
}

func TestShutdown(t *testing.T) {
	testShutdown(t, "tcp", ":9991")
}

type testShutdownServer struct {
	*BuiltinEventEngine
	tester  *testing.T
	eng     Engine
	network string
	addr    string
	count   int
	clients int32
	N       int
}

func (t *testShutdownServer) OnBoot(eng Engine) (action Action) {
	t.eng = eng
	return
}

func (t *testShutdownServer) OnOpen(Conn) (out []byte, action Action) {
	require.EqualValues(t.tester, atomic.AddInt32(&t.clients, 1), t.eng.CountConnections())
	return
}

func (t *testShutdownServer) OnClose(Conn, error) (action Action) {
	atomic.AddInt32(&t.clients, -1)
	return
}

func (t *testShutdownServer) OnTick() (delay time.Duration, action Action) {
	if t.count == 0 {
		// start clients
		for i := 0; i < t.N; i++ {
			go func() {
				conn, err := net.Dial(t.network, t.addr)
				require.NoError(t.tester, err)
				defer conn.Close()
				_, err = conn.Read([]byte{0})
				require.Error(t.tester, err)
			}()
		}
	} else if int(atomic.LoadInt32(&t.clients)) == t.N {
		action = Shutdown
	}
	t.count++
	delay = time.Second / 20
	return
}

func testShutdown(t *testing.T, network, addr string) {
	currentLogger, currentFlusher := logging.GetDefaultLogger(), logging.GetDefaultFlusher()
	t.Cleanup(func() {
		logging.SetDefaultLoggerAndFlusher(currentLogger, currentFlusher) // restore
	})

	events := &testShutdownServer{tester: t, network: network, addr: addr, N: 100}
	logPath := filepath.Join(t.TempDir(), "gnet-test-shutdown.log")
	err := Run(events, network+"://"+addr,
		WithLogPath(logPath),
		WithLogLevel(logging.WarnLevel),
		WithTicker(true),
		WithReadBufferCap(512),
		WithWriteBufferCap(512))
	assert.NoError(t, err)
	require.Equal(t, 0, int(events.clients), "did not close all clients")
}

func TestCloseActionError(t *testing.T) {
	testCloseActionError(t, "tcp", ":9992")
}

type testCloseActionErrorServer struct {
	*BuiltinEventEngine
	tester        *testing.T
	network, addr string
	action        bool
}

func (t *testCloseActionErrorServer) OnClose(Conn, error) (action Action) {
	action = Shutdown
	return
}

func (t *testCloseActionErrorServer) OnTraffic(c Conn) (action Action) {
	n := c.InboundBuffered()
	buf := make([]byte, n)
	m, err := c.Read(buf)
	assert.NoError(t.tester, err)
	assert.EqualValuesf(t.tester, n, m, "read %d bytes, expected %d", m, n)
	n, err = c.Write(buf)
	assert.NoError(t.tester, err)
	assert.EqualValuesf(t.tester, m, n, "wrote %d bytes, expected %d", n, m)
	action = Close
	return
}

func (t *testCloseActionErrorServer) OnTick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			require.NoError(t.tester, err)
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testCloseActionError(t *testing.T, network, addr string) {
	events := &testCloseActionErrorServer{tester: t, network: network, addr: addr}
	err := Run(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestShutdownActionError(t *testing.T) {
	testShutdownActionError(t, "tcp", ":9993")
}

type testShutdownActionErrorServer struct {
	*BuiltinEventEngine
	tester        *testing.T
	network, addr string
	action        bool
}

func (t *testShutdownActionErrorServer) OnTraffic(c Conn) (action Action) {
	buf, _ := c.Peek(-1)
	_, _ = c.Write(buf)
	_, _ = c.Discard(-1)
	action = Shutdown
	return
}

func (t *testShutdownActionErrorServer) OnTick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			require.NoError(t.tester, err)
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testShutdownActionError(t *testing.T, network, addr string) {
	events := &testShutdownActionErrorServer{tester: t, network: network, addr: addr}
	err := Run(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestCloseActionOnOpen(t *testing.T) {
	testCloseActionOnOpen(t, "tcp", ":9994")
}

type testCloseActionOnOpenServer struct {
	*BuiltinEventEngine
	tester        *testing.T
	network, addr string
	action        bool
}

func (t *testCloseActionOnOpenServer) OnOpen(Conn) (out []byte, action Action) {
	action = Close
	return
}

func (t *testCloseActionOnOpenServer) OnClose(Conn, error) (action Action) {
	action = Shutdown
	return
}

func (t *testCloseActionOnOpenServer) OnTick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testCloseActionOnOpen(t *testing.T, network, addr string) {
	events := &testCloseActionOnOpenServer{tester: t, network: network, addr: addr}
	err := Run(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestShutdownActionOnOpen(t *testing.T) {
	testShutdownActionOnOpen(t, "tcp", ":9995")
}

type testShutdownActionOnOpenServer struct {
	*BuiltinEventEngine
	tester        *testing.T
	network, addr string
	action        bool
	eng           Engine
}

func (t *testShutdownActionOnOpenServer) OnOpen(Conn) (out []byte, action Action) {
	action = Shutdown
	return
}

func (t *testShutdownActionOnOpenServer) OnShutdown(e Engine) {
	t.eng = e
	fd, err := t.eng.Dup()
	assert.Greaterf(t.tester, fd, 2, "expected fd: > 2, but got: %d", fd)
	require.NoErrorf(t.tester, err, "dup error")
	assert.NoErrorf(t.tester, SysClose(fd), "close error")
	logging.Debugf("dup fd: %d with error: %v\n", fd, err)
}

func (t *testShutdownActionOnOpenServer) OnTick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testShutdownActionOnOpen(t *testing.T, network, addr string) {
	events := &testShutdownActionOnOpenServer{tester: t, network: network, addr: addr}
	err := Run(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
	_, err = events.eng.Dup()
	assert.ErrorIsf(t, err, errorx.ErrEngineInShutdown, "expected error: %v, but got: %v",
		errorx.ErrEngineInShutdown, err)
}

func TestUDPShutdown(t *testing.T) {
	testUDPShutdown(t, "udp4", ":9000")
}

type testUDPShutdownServer struct {
	*BuiltinEventEngine
	tester  *testing.T
	network string
	addr    string
	tick    bool
}

func (t *testUDPShutdownServer) OnTraffic(c Conn) (action Action) {
	buf, _ := c.Peek(-1)
	_, _ = c.Write(buf)
	_, _ = c.Discard(-1)
	action = Shutdown
	return
}

func (t *testUDPShutdownServer) OnTick() (delay time.Duration, action Action) {
	if !t.tick {
		t.tick = true
		delay = time.Millisecond * 100
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, err = conn.Write(data)
			require.NoError(t.tester, err)
			_, err = conn.Read(data)
			require.NoError(t.tester, err)
		}()
		return
	}
	delay = time.Millisecond * 100
	return
}

func testUDPShutdown(t *testing.T, network, addr string) {
	svr := &testUDPShutdownServer{tester: t, network: network, addr: addr}
	err := Run(svr, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestCloseConnection(t *testing.T) {
	testCloseConnection(t, "tcp", ":9996")
}

type testCloseConnectionServer struct {
	*BuiltinEventEngine
	tester        *testing.T
	network, addr string
	action        bool
}

func (t *testCloseConnectionServer) OnClose(Conn, error) (action Action) {
	action = Shutdown
	return
}

func (t *testCloseConnectionServer) OnTraffic(c Conn) (action Action) {
	buf, _ := c.Peek(-1)
	_, _ = c.Write(buf)
	_, _ = c.Discard(-1)
	go func() {
		time.Sleep(time.Second)
		_ = c.CloseWithCallback(func(_ Conn, err error) error {
			assert.ErrorIsf(t.tester, err, errorx.ErrEngineShutdown, "should be engine shutdown error")
			return nil
		})
	}()
	return
}

func (t *testCloseConnectionServer) OnTick() (delay time.Duration, action Action) {
	delay = time.Millisecond * 100
	if !t.action {
		t.action = true
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			require.NoError(t.tester, err)
			// waiting the engine shutdown.
			_, err = conn.Read(data)
			require.Error(t.tester, err)
		}()
		return
	}
	return
}

func testCloseConnection(t *testing.T, network, addr string) {
	events := &testCloseConnectionServer{tester: t, network: network, addr: addr}
	err := Run(events, network+"://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func TestServerOptionsCheck(t *testing.T) {
	err := Run(&BuiltinEventEngine{}, "tcp://:3500", WithNumEventLoop(10001), WithLockOSThread(true))
	assert.EqualError(t, err, errorx.ErrTooManyEventLoopThreads.Error(), "error returned with LockOSThread option")
}

func TestStopServer(t *testing.T) {
	testStop(t, "tcp", ":9997")
}

type testStopServer struct {
	*BuiltinEventEngine
	tester                   *testing.T
	network, addr, protoAddr string
	eng                      Engine
	action                   bool
}

func (t *testStopServer) OnBoot(eng Engine) (action Action) {
	t.eng = eng
	return
}

func (t *testStopServer) OnClose(Conn, error) (action Action) {
	logging.Debugf("closing connection...")
	return
}

func (t *testStopServer) OnTraffic(c Conn) (action Action) {
	buf, _ := c.Peek(-1)
	_, _ = c.Write(buf)
	_, _ = c.Discard(-1)
	return
}

func (t *testStopServer) OnTick() (delay time.Duration, action Action) {
	delay = time.Millisecond * 100
	if !t.action {
		t.action = true
		go func() {
			conn, err := net.Dial(t.network, t.addr)
			require.NoError(t.tester, err)
			defer conn.Close()
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			require.NoError(t.tester, err)

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				logging.Debugf("stop engine...", t.eng.Stop(ctx))
			}()

			// waiting the engine shutdown.
			_, err = conn.Read(data)
			require.Error(t.tester, err)
		}()
		return
	}
	return
}

func testStop(t *testing.T, network, addr string) {
	events := &testStopServer{tester: t, network: network, addr: addr, protoAddr: network + "://" + addr}
	err := Run(events, events.protoAddr, WithTicker(true))
	assert.NoError(t, err)
}

func TestEngineStop(t *testing.T) {
	testEngineStop(t, "tcp", ":9998")
}

type testStopEngine struct {
	*BuiltinEventEngine
	tester                   *testing.T
	network, addr, protoAddr string
	eng                      Engine
	stopIter                 int64
	name                     string
	exchngCount              int64
}

func (t *testStopEngine) OnBoot(eng Engine) (action Action) {
	t.eng = eng
	return
}

func (t *testStopEngine) OnClose(Conn, error) (action Action) {
	logging.Debugf("closing connection...")
	return
}

func (t *testStopEngine) OnTraffic(c Conn) (action Action) {
	buf, _ := c.Peek(-1)
	_, _ = c.Write(buf)
	_, _ = c.Discard(-1)
	atomic.AddInt64(&t.exchngCount, 1)
	return
}

func (t *testStopEngine) OnTick() (delay time.Duration, action Action) {
	delay = time.Millisecond * 100
	go func() {
		conn, err := net.Dial(t.network, t.addr)
		require.NoError(t.tester, err)
		defer conn.Close()
		data := []byte("Hello World! " + t.name)
		_, _ = conn.Write(data)
		_, err = conn.Read(data)
		require.NoError(t.tester, err)

		iter := atomic.LoadInt64(&t.stopIter)
		if iter <= 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			logging.Debugf("stop engine...", t.eng.Stop(ctx))
			// waiting the engine shutdown.
			_, err = conn.Read(data)
			require.Error(t.tester, err)
		}
		atomic.AddInt64(&t.stopIter, -1)
	}()
	return
}

func testEngineStop(t *testing.T, network, addr string) {
	events1 := &testStopEngine{tester: t, network: network, addr: addr, protoAddr: network + "://" + addr, name: "1", stopIter: 2}
	events2 := &testStopEngine{tester: t, network: network, addr: addr, protoAddr: network + "://" + addr, name: "2", stopIter: 5}

	result1 := make(chan error, 1)
	go func() {
		err := Run(events1, events1.protoAddr, WithTicker(true), WithReuseAddr(true), WithReusePort(true))
		result1 <- err
	}()
	// ensure the first handler processes before starting the next since the delay per tick is 100ms
	time.Sleep(150 * time.Millisecond)
	result2 := make(chan error, 1)
	go func() {
		err := Run(events2, events2.protoAddr, WithTicker(true), WithReuseAddr(true), WithReusePort(true))
		result2 <- err
	}()

	err := <-result1
	assert.NoError(t, err)
	err = <-result2
	assert.NoError(t, err)
	// make sure that each handler processed at least 1
	require.Greater(t, events1.exchngCount, int64(0))
	require.Greater(t, events2.exchngCount, int64(0))
	require.Equal(t, int64(2+1+5+1), events1.exchngCount+events2.exchngCount)
	// stop an already stopped engine
	require.Equal(t, errorx.ErrEngineInShutdown, events1.eng.Stop(context.Background()))
}

// Test should not panic when we wake-up server_closed conn.
func TestClosedWakeUp(t *testing.T) {
	events := &testClosedWakeUpServer{
		tester:             t,
		BuiltinEventEngine: &BuiltinEventEngine{}, network: "tcp", addr: ":9999", protoAddr: "tcp://:9999",
		clientClosed: make(chan struct{}),
		serverClosed: make(chan struct{}),
		wakeup:       make(chan struct{}),
	}

	err := Run(events, events.protoAddr)
	assert.NoError(t, err)
}

type testClosedWakeUpServer struct {
	*BuiltinEventEngine
	tester                   *testing.T
	network, addr, protoAddr string

	wakeup       chan struct{}
	serverClosed chan struct{}
	clientClosed chan struct{}
}

func (s *testClosedWakeUpServer) OnBoot(eng Engine) (action Action) {
	go func() {
		c, err := net.Dial(s.network, s.addr)
		require.NoError(s.tester, err)

		_, err = c.Write([]byte("hello"))
		require.NoError(s.tester, err)

		<-s.wakeup
		_, err = c.Write([]byte("hello again"))
		require.NoError(s.tester, err)

		close(s.clientClosed)
		<-s.serverClosed

		logging.Debugf("stop engine...", eng.Stop(context.TODO()))
	}()

	return None
}

func (s *testClosedWakeUpServer) OnTraffic(c Conn) Action {
	assert.NotNil(s.tester, c.RemoteAddr())

	select {
	case <-s.wakeup:
	default:
		close(s.wakeup)
	}

	go func() { require.NoError(s.tester, c.Wake(nil)) }()
	go func() { require.NoError(s.tester, c.Close()) }()

	<-s.clientClosed

	_, _ = c.Write([]byte("answer"))
	return None
}

func (s *testClosedWakeUpServer) OnClose(Conn, error) (action Action) {
	select {
	case <-s.serverClosed:
	default:
		close(s.serverClosed)
	}
	return
}

type testMultiInstLoggerRaceServer struct {
	*BuiltinEventEngine
}

func (t *testMultiInstLoggerRaceServer) OnBoot(_ Engine) (action Action) {
	return Shutdown
}

func TestMultiInstLoggerRace(t *testing.T) {
	currentLogger, currentFlusher := logging.GetDefaultLogger(), logging.GetDefaultFlusher()
	t.Cleanup(func() {
		logging.SetDefaultLoggerAndFlusher(currentLogger, currentFlusher) // restore
	})

	logger1, _ := zap.NewDevelopment()
	events1 := new(testMultiInstLoggerRaceServer)
	g := errgroup.Group{}
	g.Go(func() error {
		err := Run(events1, "tulip://howdy", WithLogger(logger1.Sugar()))
		return err
	})

	logger2, _ := zap.NewDevelopment()
	events2 := new(testMultiInstLoggerRaceServer)
	g.Go(func() error {
		err := Run(events2, "tulip://howdy", WithLogger(logger2.Sugar()))
		return err
	})

	assert.ErrorIs(t, g.Wait(), errorx.ErrUnsupportedProtocol)
}

var errIncompletePacket = errors.New("incomplete packet")

type simServer struct {
	BuiltinEventEngine
	tester       *testing.T
	eng          Engine
	network      string
	addr         string
	multicore    bool
	nclients     int
	packetSize   int
	batchWrite   int
	batchRead    int
	started      int32
	connected    int32
	disconnected int32
}

func (s *simServer) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	return
}

func (s *simServer) OnOpen(c Conn) (out []byte, action Action) {
	c.SetContext(&testCodec{})
	atomic.AddInt32(&s.connected, 1)
	out = []byte("sweetness\r\n")
	require.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	require.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
	return
}

func (s *simServer) OnClose(_ Conn, err error) (action Action) {
	if err != nil {
		logging.Debugf("error occurred on closed, %v\n", err)
	}

	atomic.AddInt32(&s.disconnected, 1)
	if atomic.LoadInt32(&s.connected) == atomic.LoadInt32(&s.disconnected) &&
		atomic.LoadInt32(&s.disconnected) == int32(s.nclients) {
		action = Shutdown
	}

	return
}

func (s *simServer) OnTraffic(c Conn) (action Action) {
	codec := c.Context().(*testCodec)
	var packets [][]byte
	for i := 0; i < s.batchRead; i++ {
		data, err := codec.Decode(c)
		if errors.Is(err, errIncompletePacket) {
			break
		}
		if err != nil {
			logging.Errorf("invalid packet: %v", err)
			return Close
		}
		packet, _ := codec.Encode(data)
		packets = append(packets, packet)
	}
	if n := len(packets); n > 1 {
		_, _ = c.Writev(packets)
	} else if n == 1 {
		_, _ = c.Write(packets[0])
	}
	if len(packets) == s.batchRead && c.InboundBuffered() > 0 {
		err := c.Wake(nil) // wake up the connection manually to avoid missing the leftover data
		assert.NoError(s.tester, err)
	}
	return
}

func (s *simServer) OnTick() (delay time.Duration, action Action) {
	if atomic.CompareAndSwapInt32(&s.started, 0, 1) {
		for i := 0; i < s.nclients; i++ {
			go func() {
				runSimClient(s.tester, s.network, s.addr, s.packetSize, s.batchWrite)
			}()
		}
	}
	delay = 100 * time.Millisecond
	return
}

// All current protocols.
const (
	magicNumber     = 1314
	magicNumberSize = 2
	bodySize        = 4
)

var magicNumberBytes []byte

func init() {
	magicNumberBytes = make([]byte, magicNumberSize)
	binary.BigEndian.PutUint16(magicNumberBytes, uint16(magicNumber))
}

// Protocol format:
//
// * 0           2                       6
// * +-----------+-----------------------+
// * |   magic   |       body len        |
// * +-----------+-----------+-----------+
// * |                                   |
// * +                                   +
// * |           body bytes              |
// * +                                   +
// * |            ... ...                |
// * +-----------------------------------+.
type testCodec struct{}

func (codec testCodec) Encode(buf []byte) ([]byte, error) {
	bodyOffset := magicNumberSize + bodySize
	msgLen := bodyOffset + len(buf)

	data := make([]byte, msgLen)
	copy(data, magicNumberBytes)

	binary.BigEndian.PutUint32(data[magicNumberSize:bodyOffset], uint32(len(buf)))
	copy(data[bodyOffset:msgLen], buf)
	return data, nil
}

func (codec testCodec) Decode(c Conn) ([]byte, error) {
	bodyOffset := magicNumberSize + bodySize
	buf, err := c.Peek(bodyOffset)
	if err != nil {
		if errors.Is(err, io.ErrShortBuffer) {
			err = errIncompletePacket
		}
		return nil, err
	}

	if !bytes.Equal(magicNumberBytes, buf[:magicNumberSize]) {
		return nil, errors.New("invalid magic number")
	}

	bodyLen := binary.BigEndian.Uint32(buf[magicNumberSize:bodyOffset])
	msgLen := bodyOffset + int(bodyLen)
	buf, err = c.Peek(msgLen)
	if err != nil {
		if errors.Is(err, io.ErrShortBuffer) {
			err = errIncompletePacket
		}
		return nil, err
	}
	body := make([]byte, bodyLen)
	copy(body, buf[bodyOffset:msgLen])
	_, _ = c.Discard(msgLen)

	return body, nil
}

func (codec testCodec) Unpack(buf []byte) ([]byte, error) {
	bodyOffset := magicNumberSize + bodySize
	if len(buf) < bodyOffset {
		return nil, errIncompletePacket
	}

	if !bytes.Equal(magicNumberBytes, buf[:magicNumberSize]) {
		return nil, errors.New("invalid magic number")
	}

	bodyLen := binary.BigEndian.Uint32(buf[magicNumberSize:bodyOffset])
	msgLen := bodyOffset + int(bodyLen)
	if len(buf) < msgLen {
		return nil, errIncompletePacket
	}

	return buf[bodyOffset:msgLen], nil
}

func TestSimServer(t *testing.T) {
	t.Run("packet-size=64,batch=200", func(t *testing.T) {
		runSimServer(t, ":7200", true, 10, 64, 200, -1)
	})
	t.Run("packet-size=128,batch=100", func(t *testing.T) {
		runSimServer(t, ":7201", false, 10, 128, 100, 10)
	})
	t.Run("packet-size=256,batch=50", func(t *testing.T) {
		runSimServer(t, ":7202", true, 10, 256, 50, -1)
	})
	t.Run("packet-size=512,batch=30", func(t *testing.T) {
		runSimServer(t, ":7203", false, 10, 512, 30, 3)
	})
	t.Run("packet-size=1024,batch=20", func(t *testing.T) {
		runSimServer(t, ":7204", true, 10, 1024, 20, -1)
	})
	t.Run("packet-size=64*1024,batch=10", func(t *testing.T) {
		runSimServer(t, ":7205", false, 10, 64*1024, 10, 1)
	})
	t.Run("packet-size=128*1024,batch=5", func(t *testing.T) {
		runSimServer(t, ":7206", true, 10, 128*1024, 5, -1)
	})
	t.Run("packet-size=512*1024,batch=3", func(t *testing.T) {
		runSimServer(t, ":7207", false, 10, 512*1024, 3, 1)
	})
	t.Run("packet-size=1024*1024,batch=2", func(t *testing.T) {
		runSimServer(t, ":7208", true, 10, 1024*1024, 2, -1)
	})
}

func runSimServer(t *testing.T, addr string, et bool, nclients, packetSize, batchWrite, batchRead int) {
	ts := &simServer{
		tester:     t,
		network:    "tcp",
		addr:       addr,
		multicore:  true,
		nclients:   nclients,
		packetSize: packetSize,
		batchWrite: batchWrite,
		batchRead:  batchRead,
	}
	if batchRead < 0 {
		ts.batchRead = math.MaxInt32 // unlimited read batch
	}
	err := Run(ts,
		ts.network+"://"+ts.addr,
		WithEdgeTriggeredIO(et),
		WithMulticore(ts.multicore),
		WithTicker(true),
		WithTCPKeepAlive(time.Minute*1))
	assert.NoError(t, err)
}

func runSimClient(t *testing.T, network, addr string, packetSize, batch int) {
	c, err := net.Dial(network, addr)
	require.NoError(t, err)
	defer c.Close()
	rd := bufio.NewReader(c)
	msg, err := rd.ReadBytes('\n')
	require.NoError(t, err)
	require.Equal(t, string(msg), "sweetness\r\n", "bad header")
	var duration time.Duration
	packetBytes := packetSize * batch
	switch {
	case packetBytes < 16*1024:
		duration = 2 * time.Second
	case packetBytes < 32*1024:
		duration = 3 * time.Second
	case packetBytes < 480*1024:
		duration = 4 * time.Second
	default:
		duration = 5 * time.Second
	}
	logging.Debugf("test duration: %v", duration)
	start := time.Now()
	for time.Since(start) < duration {
		batchSendAndRecv(t, c, rd, packetSize, batch)
	}
}

func batchSendAndRecv(t *testing.T, c net.Conn, rd *bufio.Reader, packetSize, batch int) {
	codec := testCodec{}
	var (
		requests  [][]byte
		buf       []byte
		packetLen int
	)
	for i := 0; i < batch; i++ {
		req := make([]byte, packetSize)
		_, err := crand.Read(req)
		require.NoError(t, err)
		requests = append(requests, req)
		packet, _ := codec.Encode(req)
		packetLen = len(packet)
		buf = append(buf, packet...)
	}
	_, err := c.Write(buf)
	require.NoError(t, err)
	respPacket := make([]byte, batch*packetLen)
	_, err = io.ReadFull(rd, respPacket)
	require.NoError(t, err)
	for i, req := range requests {
		rsp, err := codec.Unpack(respPacket[i*packetLen:])
		require.NoError(t, err)
		require.Equalf(t, req, rsp, "request and response mismatch, packet size: %d, batch: %d, round: %d",
			packetSize, batch, i)
	}
}
