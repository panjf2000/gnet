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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
}

func (s *testServer) OnBoot(eng Engine) (action Action) {
	s.eng = eng
	if len(s.addrs) > 1 {
		fd, err := s.eng.Dup()
		assert.ErrorIsf(s.tester, err, errorx.ErrUnsupportedOp, "dup error")
		assert.EqualValuesf(s.tester, -1, fd, "expected fd: -1, but got: %d", fd)

		addr := s.addrs[rand.Intn(len(s.addrs))]
		network, address, _ := parseProtoAddr(addr)
		fd, err = s.eng.DupListener(network, address)
		assert.NoErrorf(s.tester, err, "DupListener error")
		assert.Greaterf(s.tester, fd, 2, "expected duplicated fd: > 2, but got: %d", fd)
		assert.NoErrorf(s.tester, SysClose(fd), "close fd error")

		// Test invalid input
		fd, err = s.eng.DupListener("tcp", "abc")
		assert.ErrorIsf(s.tester, err, errorx.ErrInvalidNetworkAddress, "expected ErrInvalidNetworkAddress")
		assert.EqualValuesf(s.tester, -1, fd, "expected fd: -1, but got: %d", fd)
	} else {
		fd, err := s.eng.Dup()
		assert.NoErrorf(s.tester, err, "Dup error")
		assert.Greaterf(s.tester, fd, 2, "expected duplicated fd: > 2, but got: %d", fd)
		assert.NoErrorf(s.tester, SysClose(fd), "close fd error")
	}
	return
}

func (s *testServer) OnOpen(c Conn) (out []byte, action Action) {
	c.SetContext(c)
	atomic.AddInt32(&s.connected, 1)
	out = []byte("andypan\r\n")
	assert.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	assert.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
	return
}

func (s *testServer) OnShutdown(_ Engine) {
	if len(s.addrs) > 1 {
		fd, err := s.eng.Dup()
		assert.ErrorIsf(s.tester, err, errorx.ErrUnsupportedOp, "dup error")
		assert.EqualValuesf(s.tester, -1, fd, "expected fd: -1, but got: %d", fd)

		addr := s.addrs[rand.Intn(len(s.addrs))]
		network, address, _ := parseProtoAddr(addr)
		fd, err = s.eng.DupListener(network, address)
		assert.NoErrorf(s.tester, err, "DupListener error")
		assert.Greaterf(s.tester, fd, 2, "expected duplicated fd: > 2, but got: %d", fd)
		assert.NoErrorf(s.tester, SysClose(fd), "close fd error")

		// Test invalid input
		fd, err = s.eng.DupListener("tcp", "abc")
		assert.ErrorIsf(s.tester, err, errorx.ErrInvalidNetworkAddress, "expected ErrInvalidNetworkAddress")
		assert.EqualValuesf(s.tester, -1, fd, "expected fd: -1, but got: %d", fd)
	} else {
		fd, err := s.eng.Dup()
		assert.NoErrorf(s.tester, err, "Dup error")
		assert.Greaterf(s.tester, fd, 2, "expected duplicated fd: > 2, but got: %d", fd)
		assert.NoErrorf(s.tester, SysClose(fd), "close fd error")
	}
}

func (s *testServer) OnClose(c Conn, err error) (action Action) {
	if err != nil {
		logging.Debugf("error occurred on closed, %v\n", err)
	}

	assert.Equal(s.tester, c.Context(), c, "invalid context")

	atomic.AddInt32(&s.disconnected, 1)
	return
}

func (s *testServer) OnTraffic(c Conn) (action Action) {
	if s.async {
		buf := bbPool.Get()
		n, err := c.WriteTo(buf)
		assert.NoError(s.tester, err, "WriteTo error")
		assert.Greater(s.tester, n, int64(0), "WriteTo error")
		if c.LocalAddr().Network() == "tcp" || c.LocalAddr().Network() == "unix" {
			err = goPool.DefaultWorkerPool.Submit(
				func() {
					if s.writev {
						mid := buf.Len() / 2
						bs := make([][]byte, 2)
						bs[0] = buf.B[:mid]
						bs[1] = buf.B[mid:]
						err := c.AsyncWritev(bs, func(c Conn, err error) error {
							if c.RemoteAddr() != nil {
								logging.Debugf("conn=%s done writev: %v", c.RemoteAddr().String(), err)
							}
							bbPool.Put(buf)
							return nil
						})
						assert.NoError(s.tester, err, "AsyncWritev error")
					} else {
						err := c.AsyncWrite(buf.Bytes(), func(c Conn, err error) error {
							if c.RemoteAddr() != nil {
								logging.Debugf("conn=%s done write: %v", c.RemoteAddr().String(), err)
							}
							bbPool.Put(buf)
							return nil
						})
						assert.NoError(s.tester, err, "AsyncWritev error")
					}
				})
			assert.NoError(s.tester, err)
			return
		} else if c.LocalAddr().Network() == "udp" {
			err := goPool.DefaultWorkerPool.Submit(
				func() {
					err := c.AsyncWrite(buf.Bytes(), nil)
					assert.NoError(s.tester, err, "AsyncWritev error")
				})
			assert.NoError(s.tester, err)
			return
		}
		return
	}

	buf, err := c.Next(-1)
	assert.NoError(s.tester, err, "reading data error")
	var n int
	if s.writev {
		mid := len(buf) / 2
		n, err = c.Writev([][]byte{buf[:mid], buf[mid:]})
	} else {
		n, err = c.Write(buf)
	}
	assert.NoError(s.tester, err, "writev error")
	assert.EqualValues(s.tester, n, len(buf), "short writev")

	// Only for code coverage of testing.
	if !s.multicore {
		assert.NoErrorf(s.tester, c.Flush(), "flush error")
		_ = c.Fd()
		fd, err := c.Dup()
		assert.NoErrorf(s.tester, err, "dup error")
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

		assert.Zero(s.tester, c.InboundBuffered(), "inbound buffer error")
		assert.GreaterOrEqual(s.tester, c.OutboundBuffered(), 0, "outbound buffer error")
		n, err := c.Discard(1)
		assert.NoErrorf(s.tester, err, "discard error")
		assert.Zerof(s.tester, n, "discard error")
		assert.ErrorIs(s.tester, c.SetDeadline(time.Now().Add(time.Second)), errorx.ErrUnsupportedOp)
		assert.ErrorIs(s.tester, c.SetReadDeadline(time.Now().Add(time.Second)), errorx.ErrUnsupportedOp)
		assert.ErrorIs(s.tester, c.SetWriteDeadline(time.Now().Add(time.Second)), errorx.ErrUnsupportedOp)

		if c.LocalAddr().Network() == "udp" {
			n, err := c.Writev([][]byte{})
			assert.ErrorIs(s.tester, err, errorx.ErrUnsupportedOp, "udp Writev error")
			assert.Zero(s.tester, n, "udp Writev error")
			err = c.AsyncWritev([][]byte{}, nil)
			assert.ErrorIs(s.tester, err, errorx.ErrUnsupportedOp, "udp Writev error")
			_, err = c.SendTo(buf, nil)
			assert.ErrorIsf(s.tester, err, errorx.ErrInvalidNetworkAddress,
				"got error: %v, expected error: %v", err, errorx.ErrInvalidNetworkAddress)
		} else {
			_, err = c.SendTo(buf, c.RemoteAddr())
			assert.ErrorIsf(s.tester, err, errorx.ErrUnsupportedOp,
				"got error: %v, expected error: %v", err, errorx.ErrUnsupportedOp)
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
				err := goPool.DefaultWorkerPool.Submit(func() {
					startClient(s.tester, proto, addr, s.multicore, s.async, 0, nil)
					atomic.AddInt32(&s.clientActive, -1)
				})
				assert.NoError(s.tester, err)
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
			assert.EqualValues(s.tester, 0, s.eng.CountConnections())
		}
	}
	return
}

func runServer(t *testing.T, addrs []string, conf *testConf) {
	ts := &testServer{
		tester:    t,
		addrs:     addrs,
		multicore: conf.multicore,
		async:     conf.async,
		writev:    conf.writev,
		nclients:  conf.clients,
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

func startClient(t *testing.T, network, addr string, multicore, async bool, packetSize int, stallCh chan struct{}) {
	c, err := net.Dial(network, addr)
	assert.NoError(t, err)
	defer c.Close() //nolint:errcheck
	rd := bufio.NewReader(c)
	if network != "udp" {
		msg, err := rd.ReadBytes('\n')
		assert.NoError(t, err)
		assert.Equal(t, string(msg), "andypan\r\n", "bad header")
	}

	if stallCh != nil {
		<-stallCh
	}

	duration := time.Duration((rand.Float64()*2+1)*float64(time.Second)) / 2
	logging.Debugf("test duration: %v", duration)
	start := time.Now()
	if packetSize == 0 {
		packetSize = streamLen
		if network == "udp" {
			packetSize = datagramLen
		}
	}
	for time.Since(start) < duration {
		reqData := make([]byte, packetSize)
		_, err = crand.Read(reqData)
		assert.NoError(t, err)
		_, err = c.Write(reqData)
		assert.NoError(t, err)
		respData := make([]byte, len(reqData))
		_, err = io.ReadFull(rd, respData)
		assert.NoError(t, err)
		if !async {
			// assert.Equalf(t, reqData, respData, "response mismatch with protocol:%s, multi-core:%t, content of bytes: %d vs %d", network, multicore, string(reqData), string(respData))
			assert.Equalf(
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
	require.ErrorIs(t, err, errorx.ErrUnsupportedProtocol)
	err = Run(events, "howdy")
	require.ErrorIs(t, err, errorx.ErrInvalidNetworkAddress)
	err = Run(events, "tcp://")
	require.ErrorIs(t, err, errorx.ErrInvalidNetworkAddress)
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
	require.NoError(t, err)
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
		err := goPool.DefaultWorkerPool.Submit(func() {
			conn, err := net.Dial(t.network, t.addr)
			assert.NoError(t.tester, err)
			defer conn.Close() //nolint:errcheck
			r := make([]byte, 10)
			_, err = conn.Read(r)
			assert.NoError(t.tester, err)
		})
		assert.NoError(t.tester, err)
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
	assert.EqualValues(t.tester, atomic.AddInt32(&t.clients, 1), t.eng.CountConnections())
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
			err := goPool.DefaultWorkerPool.Submit(func() {
				conn, err := net.Dial(t.network, t.addr)
				assert.NoError(t.tester, err)
				defer conn.Close() //nolint:errcheck
				_, err = conn.Read([]byte{0})
				assert.Error(t.tester, err)
			})
			assert.NoError(t.tester, err)
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
	assert.Equal(t, 0, int(events.clients), "did not close all clients")
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
		err := goPool.DefaultWorkerPool.Submit(func() {
			conn, err := net.Dial(t.network, t.addr)
			assert.NoError(t.tester, err)
			defer conn.Close() //nolint:errcheck
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			assert.NoError(t.tester, err)
		})
		assert.NoError(t.tester, err)
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
		err := goPool.DefaultWorkerPool.Submit(func() {
			conn, err := net.Dial(t.network, t.addr)
			assert.NoError(t.tester, err)
			defer conn.Close() //nolint:errcheck
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			assert.NoError(t.tester, err)
		})
		assert.NoError(t.tester, err)
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
		err := goPool.DefaultWorkerPool.Submit(func() {
			conn, err := net.Dial(t.network, t.addr)
			assert.NoError(t.tester, err)
			defer conn.Close() //nolint:errcheck
		})
		assert.NoError(t.tester, err)
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
	assert.NoErrorf(t.tester, err, "dup error")
	assert.NoErrorf(t.tester, SysClose(fd), "close error")
	logging.Debugf("dup fd: %d with error: %v\n", fd, err)
}

func (t *testShutdownActionOnOpenServer) OnTick() (delay time.Duration, action Action) {
	if !t.action {
		t.action = true
		delay = time.Millisecond * 100
		err := goPool.DefaultWorkerPool.Submit(func() {
			conn, err := net.Dial(t.network, t.addr)
			assert.NoError(t.tester, err)
			defer conn.Close() //nolint:errcheck
		})
		assert.NoError(t.tester, err)
		return
	}
	delay = time.Millisecond * 100
	return
}

func testShutdownActionOnOpen(t *testing.T, network, addr string) {
	events := &testShutdownActionOnOpenServer{tester: t, network: network, addr: addr}
	err := Run(events, network+"://"+addr, WithTicker(true))
	require.NoError(t, err)
	_, err = events.eng.Dup()
	require.ErrorIsf(t, err, errorx.ErrEngineInShutdown, "expected error: %v, but got: %v",
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
		err := goPool.DefaultWorkerPool.Submit(func() {
			conn, err := net.Dial(t.network, t.addr)
			assert.NoError(t.tester, err)
			defer conn.Close() //nolint:errcheck
			data := []byte("Hello World!")
			_, err = conn.Write(data)
			assert.NoError(t.tester, err)
			_, err = conn.Read(data)
			assert.NoError(t.tester, err)
		})
		assert.NoError(t.tester, err)
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
	err := goPool.DefaultWorkerPool.Submit(func() {
		time.Sleep(time.Second)
		err := c.CloseWithCallback(func(_ Conn, err error) error {
			assert.ErrorIsf(t.tester, err, errorx.ErrEngineShutdown, "should be engine shutdown error")
			return nil
		})
		assert.NoError(t.tester, err)
	})
	assert.NoError(t.tester, err)
	return
}

func (t *testCloseConnectionServer) OnTick() (delay time.Duration, action Action) {
	delay = time.Millisecond * 100
	if !t.action {
		t.action = true
		err := goPool.DefaultWorkerPool.Submit(func() {
			conn, err := net.Dial(t.network, t.addr)
			assert.NoError(t.tester, err)
			defer conn.Close() //nolint:errcheck
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			assert.NoError(t.tester, err)
			// waiting the engine shutdown.
			_, err = conn.Read(data)
			assert.Error(t.tester, err)
		})
		assert.NoError(t.tester, err)
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
	assert.ErrorIs(t, err, errorx.ErrTooManyEventLoopThreads, "error returned with LockOSThread option")
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
		err := goPool.DefaultWorkerPool.Submit(func() {
			conn, err := net.Dial(t.network, t.addr)
			assert.NoError(t.tester, err)
			defer conn.Close() //nolint:errcheck
			data := []byte("Hello World!")
			_, _ = conn.Write(data)
			_, err = conn.Read(data)
			assert.NoError(t.tester, err)

			err = goPool.DefaultWorkerPool.Submit(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				logging.Debugf("stop engine...", t.eng.Stop(ctx))
			})
			assert.NoError(t.tester, err)

			// waiting the engine shutdown.
			_, err = conn.Read(data)
			assert.Error(t.tester, err)
		})
		assert.NoError(t.tester, err)
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
	err := goPool.DefaultWorkerPool.Submit(func() {
		conn, err := net.Dial(t.network, t.addr)
		assert.NoError(t.tester, err)
		defer conn.Close() //nolint:errcheck
		data := []byte("Hello World! " + t.name)
		_, _ = conn.Write(data)
		_, err = conn.Read(data)
		assert.NoError(t.tester, err)

		iter := atomic.LoadInt64(&t.stopIter)
		if iter <= 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			logging.Debugf("stop engine...", t.eng.Stop(ctx))
			// waiting the engine shutdown.
			_, err = conn.Read(data)
			assert.Error(t.tester, err)
		}
		atomic.AddInt64(&t.stopIter, -1)
	})
	assert.NoError(t.tester, err)
	return
}

func testEngineStop(t *testing.T, network, addr string) {
	events1 := &testStopEngine{tester: t, network: network, addr: addr, protoAddr: network + "://" + addr, name: "1", stopIter: 2}
	events2 := &testStopEngine{tester: t, network: network, addr: addr, protoAddr: network + "://" + addr, name: "2", stopIter: 5}

	result1 := make(chan error, 1)
	err := goPool.DefaultWorkerPool.Submit(func() {
		err := Run(events1, events1.protoAddr, WithTicker(true), WithReuseAddr(true), WithReusePort(true))
		result1 <- err
	})
	require.NoError(t, err)
	// ensure the first handler processes before starting the next since the delay per tick is 100ms
	time.Sleep(150 * time.Millisecond)
	result2 := make(chan error, 1)
	err = goPool.DefaultWorkerPool.Submit(func() {
		err := Run(events2, events2.protoAddr, WithTicker(true), WithReuseAddr(true), WithReusePort(true))
		result2 <- err
	})
	require.NoError(t, err)

	err = <-result1
	require.NoError(t, err)
	err = <-result2
	require.NoError(t, err)
	// make sure that each handler processed at least 1
	require.Greater(t, events1.exchngCount, int64(0))
	require.Greater(t, events2.exchngCount, int64(0))
	require.Equal(t, int64(2+1+5+1), events1.exchngCount+events2.exchngCount)
	// stop an already stopped engine
	require.Equal(t, errorx.ErrEngineInShutdown, events1.eng.Stop(context.Background()))
}

// Test should not panic when we wake up server_closed conn.
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
	err := goPool.DefaultWorkerPool.Submit(func() {
		c, err := net.Dial(s.network, s.addr)
		assert.NoError(s.tester, err)

		_, err = c.Write([]byte("hello"))
		assert.NoError(s.tester, err)

		<-s.wakeup
		_, err = c.Write([]byte("hello again"))
		assert.NoError(s.tester, err)

		close(s.clientClosed)
		<-s.serverClosed

		logging.Debugf("stop engine...", eng.Stop(context.TODO()))
	})
	assert.NoError(s.tester, err)

	return None
}

func (s *testClosedWakeUpServer) OnTraffic(c Conn) Action {
	assert.NotNil(s.tester, c.RemoteAddr())

	select {
	case <-s.wakeup:
	default:
		close(s.wakeup)
	}

	err := goPool.DefaultWorkerPool.Submit(func() { assert.NoError(s.tester, c.Wake(nil)) })
	assert.NoError(s.tester, err)
	err = goPool.DefaultWorkerPool.Submit(func() { assert.NoError(s.tester, c.Close()) })
	assert.NoError(s.tester, err)

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

type testDisconnectedAsyncWriteServer struct {
	BuiltinEventEngine
	tester                *testing.T
	addr                  string
	writev, clientStarted bool
	exit                  atomic.Bool
}

func (t *testDisconnectedAsyncWriteServer) OnTraffic(c Conn) Action {
	_, err := c.Next(0)
	assert.NoErrorf(t.tester, err, "c.Next error: %v", err)

	err = goPool.DefaultWorkerPool.Submit(func() {
		for range time.Tick(100 * time.Millisecond) {
			if t.exit.Load() {
				break
			}

			var err error
			if t.writev {
				err = c.AsyncWritev([][]byte{[]byte("hello"), []byte("hello")}, func(_ Conn, err error) error {
					if err == nil {
						return nil
					}

					assert.ErrorIsf(t.tester, err, net.ErrClosed, "expected error: %v, but got: %v", net.ErrClosed, err)
					t.exit.Store(true)
					return nil
				})
			} else {
				err = c.AsyncWrite([]byte("hello"), func(_ Conn, err error) error {
					if err == nil {
						return nil
					}

					assert.ErrorIsf(t.tester, err, net.ErrClosed, "expected error: %v, but got: %v", net.ErrClosed, err)
					t.exit.Store(true)
					return nil
				})
			}

			if err != nil {
				return
			}
		}
	})
	assert.NoError(t.tester, err)

	return None
}

func (t *testDisconnectedAsyncWriteServer) OnTick() (delay time.Duration, action Action) {
	delay = 500 * time.Millisecond

	if t.exit.Load() {
		action = Shutdown
		return
	}

	if !t.clientStarted {
		t.clientStarted = true
		err := goPool.DefaultWorkerPool.Submit(func() {
			c, err := net.Dial("tcp", t.addr)
			assert.NoError(t.tester, err)
			_, err = c.Write([]byte("hello"))
			assert.NoError(t.tester, err)
			assert.NoError(t.tester, c.Close())
		})
		assert.NoError(t.tester, err)
	}
	return
}

func TestDisconnectedAsyncWrite(t *testing.T) {
	t.Run("async-write", func(t *testing.T) {
		events := &testDisconnectedAsyncWriteServer{tester: t, addr: ":10000"}
		err := Run(events, "tcp://:10000", WithTicker(true))
		assert.NoError(t, err)
	})
	t.Run("async-writev", func(t *testing.T) {
		events := &testDisconnectedAsyncWriteServer{tester: t, addr: ":10001", writev: true}
		err := Run(events, "tcp://:10001", WithTicker(true))
		assert.NoError(t, err)
	})
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
	out = []byte("andypan\r\n")
	assert.NotNil(s.tester, c.LocalAddr(), "nil local addr")
	assert.NotNil(s.tester, c.RemoteAddr(), "nil remote addr")
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
			err := goPool.DefaultWorkerPool.Submit(func() {
				runSimClient(s.tester, s.network, s.addr, s.packetSize, s.batchWrite)
			})
			assert.NoError(s.tester, err)
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
	assert.NoError(t, err)
	defer c.Close() //nolint:errcheck
	rd := bufio.NewReader(c)
	msg, err := rd.ReadBytes('\n')
	assert.NoError(t, err)
	assert.Equal(t, string(msg), "andypan\r\n", "bad header")
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
		assert.NoError(t, err)
		requests = append(requests, req)
		packet, _ := codec.Encode(req)
		packetLen = len(packet)
		buf = append(buf, packet...)
	}
	_, err := c.Write(buf)
	assert.NoError(t, err)
	respPacket := make([]byte, batch*packetLen)
	_, err = io.ReadFull(rd, respPacket)
	assert.NoError(t, err)
	for i, req := range requests {
		rsp, err := codec.Unpack(respPacket[i*packetLen:])
		assert.NoError(t, err)
		assert.Equalf(t, req, rsp, "request and response mismatch, packet size: %d, batch: %d, round: %d",
			packetSize, batch, i)
	}
}

type testUDPSendtoServer struct {
	BuiltinEventEngine

	addr string

	tester           *testing.T
	startClientsOnce sync.Once
	broadcastMsg     []byte

	mu          sync.Mutex
	clientAddrs []net.Addr
	clientCount int
}

func (t *testUDPSendtoServer) OnBoot(_ Engine) (action Action) {
	t.broadcastMsg = []byte("Broadcasting message")
	t.clientCount = 10
	return
}

func (t *testUDPSendtoServer) OnTraffic(c Conn) Action {
	msg, err := c.Next(-1)
	assert.NoErrorf(t.tester, err, "c.Next error: %v", err)
	assert.NotZero(t.tester, msg, "c.Next should not return empty buffer")

	t.mu.Lock()
	t.clientAddrs = append(t.clientAddrs, c.RemoteAddr())
	if len(t.clientAddrs) == t.clientCount {
		for _, addr := range t.clientAddrs {
			n, err := c.SendTo(t.broadcastMsg, addr)
			assert.NoError(t.tester, err, "c.SendTo error")
			assert.EqualValuesf(t.tester, len(t.broadcastMsg), n,
				"c.SendTo should send %d bytes, but sent %d bytes", len(t.broadcastMsg), n)
		}
	}
	t.mu.Unlock()

	return None
}

func (t *testUDPSendtoServer) OnTick() (delay time.Duration, action Action) {
	t.startClientsOnce.Do(func() {
		for i := 0; i < t.clientCount; i++ {
			err := goPool.DefaultWorkerPool.Submit(func() {
				c, err := net.Dial("udp", t.addr)
				assert.NoError(t.tester, err)
				defer c.Close() //nolint:errcheck
				_, err = c.Write([]byte("Hello World!"))
				assert.NoError(t.tester, err)
				msg := make([]byte, len(t.broadcastMsg))
				_, err = c.Read(msg)
				assert.NoError(t.tester, err)
				assert.EqualValuesf(t.tester, msg, t.broadcastMsg,
					"broadcast message mismatch, expected: %s, got: %s", t.broadcastMsg, msg)
				t.mu.Lock()
				t.clientCount--
				t.mu.Unlock()
			})
			assert.NoError(t.tester, err)
		}
	})

	t.mu.Lock()
	if t.clientCount == 0 {
		action = Shutdown
	}
	t.mu.Unlock()

	delay = time.Millisecond * 100

	return
}

func TestUDPSendtoServer(t *testing.T) {
	// The listening address without an explicit IP works on Linux and Windows
	// while sendto fails on macOS with EINVAL (invalid argument) that might
	// also occur on more BSD systems: FreeBSD, OpenBSD, NetBSD, and DragonflyBSD.
	// To pass this test on all platforms, specify an explicit IP address here.
	// addr := ":10000"
	addr := "127.0.0.1:10000"
	events := &testUDPSendtoServer{tester: t, addr: addr}
	err := Run(events, "udp://"+addr, WithTicker(true))
	assert.NoError(t, err)
}

func startUDPEchoServer(t *testing.T, c *net.UDPConn) {
	defer c.Close() //nolint:errcheck

	buf := make([]byte, datagramLen)
	for {
		nr, remote, err := c.ReadFromUDP(buf)
		if err != nil {
			break
		}
		if nr > 0 {
			nw, err := c.WriteToUDP(buf[:nr], remote)
			assert.EqualValuesf(t, nr, nw, "UDP echo server should send %d bytes, but sent %d bytes", nr, nw)
			assert.NoErrorf(t, err, "error occurred on UDP echo server %v write to %s, %v",
				remote, c.LocalAddr().String(), err)
		}
	}
}

func startStreamEchoServer(t *testing.T, ln net.Listener) {
	defer func() {
		ln.Close() //nolint:errcheck
		if strings.HasPrefix(ln.Addr().Network(), "unix") {
			os.Remove(ln.Addr().String()) //nolint:errcheck
		}
	}()

	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}

		err = goPool.DefaultWorkerPool.Submit(func() {
			defer c.Close() //nolint:errcheck

			buf := make([]byte, streamLen)
			for {
				nr, err := c.Read(buf)
				if err != nil {
					break
				}
				b := buf[:nr]
				for len(b) > 0 {
					nw, err := c.Write(b)
					assert.NoErrorf(t, err, "error occurred on TCP echo server %s write to %s, %v",
						ln.Addr().String(), c.RemoteAddr().String(), err)
					b = b[nw:]
				}
			}
		})
		assert.NoError(t, err, "error occurred on TCP echo server %s submit task, %v",
			ln.Addr().String(), err)
	}
}

type streamProxyServer struct {
	BuiltinEventEngine

	engine    Engine
	eventLoop EventLoop

	stallCh chan struct{}

	tester       *testing.T
	ListenerNet  string
	ListenerAddr string

	connected          int32
	disconnected       int32
	backendEstablished int32

	backendServerPoolMu sync.Mutex
	backendServerPool   []string

	startClintOnce sync.Once

	backendServers []string
	packetSize     int
}

func (p *streamProxyServer) OnShutdown(eng Engine) {
	p.engine = eng
	p.tester.Logf("proxy server %s shutdown!", p.ListenerAddr)
}

func (p *streamProxyServer) OnOpen(c Conn) (out []byte, action Action) {
	if c.LocalAddr().String() == p.ListenerAddr { // it's a server connection
		out = []byte("andypan\r\n")
		p.backendServerPoolMu.Lock()
		backendServer := p.backendServerPool[len(p.backendServerPool)-1]
		p.backendServerPool = p.backendServerPool[:len(p.backendServerPool)-1]
		p.backendServerPoolMu.Unlock()

		// Test the error handling of the methods of EventLoop.
		_, err := c.EventLoop().Register(context.Background(), nil)
		assert.ErrorIsf(p.tester, err, errorx.ErrInvalidNetworkAddress, "Expected error: %v, but got: %v",
			errorx.ErrInvalidNetworkAddress, err)
		_, err = c.EventLoop().Enroll(context.Background(), nil)
		assert.ErrorIsf(p.tester, err, errorx.ErrInvalidNetConn, "Expected error: %v, but got: %v",
			errorx.ErrInvalidNetConn, err)
		err = c.EventLoop().Execute(context.Background(), nil)
		assert.ErrorIsf(p.tester, err, errorx.ErrNilRunnable, "Expected error: %v, but got: %v",
			errorx.ErrNilRunnable, err)
		err = c.EventLoop().Schedule(context.Background(), nil, time.Millisecond)
		assert.ErrorIsf(p.tester, err, errorx.ErrUnsupportedOp, "Expected error: %v, but got: %v",
			errorx.ErrUnsupportedOp, err)

		network, addr, err := parseProtoAddr(backendServer)
		assert.NoError(p.tester, err, "parseProtoAddr error")
		var address net.Addr
		switch {
		case strings.HasPrefix(network, "tcp"):
			address, err = net.ResolveTCPAddr(network, addr)
		case strings.HasPrefix(network, "udp"):
			address, err = net.ResolveUDPAddr(network, addr)
		case strings.HasPrefix(network, "unix"):
			address, err = net.ResolveUnixAddr(network, addr)
		default:
			assert.Failf(p.tester, "unsupported protocol", "unsupported protocol: %s", network)
		}
		assert.NoError(p.tester, err, "ResolveNetAddr error")
		ctx := NewContext(context.Background(), c)
		resCh, err := c.EventLoop().Register(ctx, address)
		assert.NoError(p.tester, err, "Register connection error")
		err = goPool.DefaultWorkerPool.Submit(func() {
			res := <-resCh
			assert.NoError(p.tester, res.Err, "Register connection error")
			assert.NotNil(p.tester, res.Conn, "Register connection nil")
		})
		assert.NoError(p.tester, err, "Submit task error")
		atomic.AddInt32(&p.connected, 1)
	} else { // it's a client connection
		// Store the client connection in the context of the server connection.
		serverConn, ok := c.Context().(Conn)
		assert.True(p.tester, ok, "context is not Conn")
		assert.NotNil(p.tester, serverConn, "context is not Conn")
		serverConn.SetContext(c)

		err := c.EventLoop().Execute(NewContext(context.Background(), c.LocalAddr()),
			RunnableFunc(func(ctx context.Context) error {
				p.tester.Logf("backend connection %v established", FromContext(ctx))
				return nil
			}))
		assert.NoError(p.tester, err, "Execute task error")

		if int(atomic.AddInt32(&p.backendEstablished, 1)) == len(p.backendServers) {
			// Unlock the clients to start sending data.
			close(p.stallCh)
		}
	}

	return
}

func (p *streamProxyServer) OnClose(c Conn, err error) (action Action) {
	connType := "client"
	if c.LocalAddr().String() == p.ListenerAddr {
		connType = "server"
	}
	logging.Debugf("closing %s connection: %s", connType, c.LocalAddr().String())

	if err != nil {
		logging.Debugf("error occurred on server connnection closing, %v", err)
	}

	if c.LocalAddr().String() == p.ListenerAddr { // it's a server connection
		pc, ok := c.Context().(Conn)
		assert.True(p.tester, ok, "server context connection is nil")
		assert.NotNil(p.tester, pc, "server context connection is nil")
		err := c.EventLoop().Close(pc) // close the corresponding client connection
		assert.NoError(p.tester, err, "Close client connection error")

		if disconnected := atomic.AddInt32(&p.disconnected, 1); int(disconnected) == len(p.backendServers) &&
			disconnected == atomic.LoadInt32(&p.connected) {
			p.eventLoop = c.EventLoop()
			action = Shutdown
		}
	}

	return
}

func (p *streamProxyServer) OnTraffic(c Conn) Action {
	connType := "client"
	if c.LocalAddr().String() == p.ListenerAddr {
		connType = "server"
	}

	pc, ok := c.Context().(Conn)
	if !ok {
		// The backend connection is not established yet, retry later.
		assert.NoError(p.tester, c.Wake(nil), "Wake connection error")
		return None
	}

	_, err := c.WriteTo(pc)
	assert.NoErrorf(p.tester, err, "%s: Write error from %s to %s",
		connType, c.LocalAddr().String(), pc.RemoteAddr().String())
	return None
}

func (p *streamProxyServer) OnTick() (time.Duration, Action) {
	p.startClintOnce.Do(func() {
		for i := 0; i < len(p.backendServers); i++ {
			err := goPool.DefaultWorkerPool.Submit(func() {
				startClient(p.tester, p.ListenerNet, p.ListenerAddr,
					true, false, p.packetSize, p.stallCh)
			})
			assert.NoErrorf(p.tester, err, "Submit backend server %s error: %v", p.ListenerAddr, err)
		}
	})
	return time.Millisecond * 200, None
}

func TestStreamProxyServer(t *testing.T) {
	t.Run("tcp-proxy-server", func(t *testing.T) {
		addr := "tcp://127.0.0.1:10000"
		backendServers := []string{
			"tcp://127.0.0.1:10001",
			"tcp://127.0.0.1:10002",
			"tcp://127.0.0.1:10003",
			"tcp://127.0.0.1:10004",
			"tcp://127.0.0.1:10005",
			"tcp://127.0.0.1:10006",
			"tcp://127.0.0.1:10007",
			"tcp://127.0.0.1:10008",
			"tcp://127.0.0.1:10009",
			"tcp://127.0.0.1:10010",
		}
		t.Run("1-loop-LT", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, false, false)
		})
		t.Run("1-loop-ET", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, false, true)
		})
		t.Run("N-loop-LT", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, true, false)
		})
		t.Run("N-loop-ET", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, true, true)
		})
	})

	t.Run("unix-proxy-server", func(t *testing.T) {
		addr := "unix://unix-proxy-server.sock"
		backendServers := []string{
			"unix://unix-proxy-server-1.sock",
			"unix://unix-proxy-server-2.sock",
			"unix://unix-proxy-server-3.sock",
			"unix://unix-proxy-server-4.sock",
			"unix://unix-proxy-server-5.sock",
			"unix://unix-proxy-server-6.sock",
			"unix://unix-proxy-server-7.sock",
			"unix://unix-proxy-server-8.sock",
			"unix://unix-proxy-server-9.sock",
			"unix://unix-proxy-server-10.sock",
		}
		t.Run("1-loop-LT", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, false, false)
		})
		t.Run("1-loop-ET", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, false, true)
		})
		t.Run("N-loop-LT", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, true, false)
		})
		t.Run("N-loop-ET", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, true, true)
		})
	})

	t.Run("udp-proxy-server", func(t *testing.T) {
		addr := "tcp://127.0.0.1:11000"
		backendServers := []string{
			"udp://127.0.0.1:11001",
			"udp://127.0.0.1:11002",
			"udp://127.0.0.1:11003",
			"udp://127.0.0.1:11004",
			"udp://127.0.0.1:11005",
			"udp://127.0.0.1:11006",
			"udp://127.0.0.1:11007",
			"udp://127.0.0.1:11008",
			"udp://127.0.0.1:11009",
			"udp://127.0.0.1:11010",
		}
		t.Run("1-loop-LT", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, false, false)
		})
		t.Run("1-loop-ET", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, false, true)
		})
		t.Run("N-loop-LT", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, true, false)
		})
		t.Run("N-loop-ET", func(t *testing.T) {
			testStreamProxyServer(t, addr, backendServers, true, true)
		})
	})
}

func testStreamProxyServer(t *testing.T, addr string, backendServers []string, multicore, et bool) {
	network, address, err := parseProtoAddr(addr)
	require.NoError(t, err, "parseProtoAddr error")

	srv := streamProxyServer{
		tester:            t,
		stallCh:           make(chan struct{}),
		ListenerNet:       network,
		ListenerAddr:      address,
		backendServers:    backendServers,
		backendServerPool: backendServers,
	}

	var (
		backends   errgroup.Group
		netServers []io.Closer
	)
	for _, backendServer := range backendServers {
		network, addr, err := parseProtoAddr(backendServer)
		require.NoError(t, err, "parseProtoAddr error")
		if strings.HasPrefix(network, "udp") {
			udpAddr, err := net.ResolveUDPAddr(network, addr)
			require.NoError(t, err, "ResolveUDPAddr error")
			c, err := net.ListenUDP("udp", udpAddr)
			require.NoError(t, err, "ListenUDP error")
			backends.Go(func() error {
				startUDPEchoServer(t, c)
				return nil
			})
			require.NoErrorf(t, err, "Start backend UDP server %s error: %v", backendServer, err)
			srv.packetSize = datagramLen
			netServers = append(netServers, c)
		} else {
			ln, err := net.Listen(network, addr)
			require.NoErrorf(t, err, "Create backend server %s error: %v", network+"://"+addr, err)
			backends.Go(func() error {
				startStreamEchoServer(t, ln)
				return nil
			})
			require.NoErrorf(t, err, "Start backend stream server %s error: %v", backendServer, err)
			srv.packetSize = streamLen
			netServers = append(netServers, ln)
		}
	}

	// Give the backend servers some time to start.
	time.Sleep(time.Second)

	err = Run(&srv, addr, WithEdgeTriggeredIO(et), WithMulticore(multicore), WithTicker(true))
	require.NoErrorf(t, err, "Run error: %v", err)

	// Test the error handling of the methods of EventLoop after a shutdown.
	_, err = srv.engine.Register(context.Background())
	assert.ErrorIsf(t, err, errorx.ErrEngineInShutdown, "Expected error: %v, but got: %v",
		errorx.ErrEngineInShutdown, err)
	_, err = srv.eventLoop.Register(context.Background(), nil)
	require.ErrorIsf(t, err, errorx.ErrEngineInShutdown, "Expected error: %v, but got: %v",
		errorx.ErrEngineInShutdown, err)
	_, err = srv.eventLoop.Enroll(context.Background(), nil)
	require.ErrorIsf(t, err, errorx.ErrEngineInShutdown, "Expected error: %v, but got: %v",
		errorx.ErrEngineInShutdown, err)
	err = srv.eventLoop.Execute(context.Background(), nil)
	require.ErrorIsf(t, err, errorx.ErrEngineInShutdown, "Expected error: %v, but got: %v",
		errorx.ErrEngineInShutdown, err)
	err = srv.eventLoop.Schedule(context.Background(), nil, time.Millisecond)
	require.ErrorIsf(t, err, errorx.ErrUnsupportedOp, "Expected error: %v, but got: %v",
		errorx.ErrUnsupportedOp, err)

	for _, server := range netServers {
		require.NoError(t, server.Close(), "Close backend server error")
	}

	backends.Wait() //nolint:errcheck
}

type udpProxyServer struct {
	BuiltinEventEngine

	packet []byte

	engine Engine

	tester *testing.T

	stallCh chan struct{}

	ListenerNet  string
	ListenerAddr string

	activeClients      int32
	backendEstablished int32
	backendBytes       int32

	initBackendPoolOnce sync.Once
	backendServerPoolMu sync.Mutex
	backendServerPool   []Conn

	startClientOnce sync.Once
	backendServers  []string
	packetSize      int
}

func (p *udpProxyServer) OnBoot(eng Engine) (action Action) {
	p.engine = eng
	_, err := eng.Register(context.Background())
	assert.ErrorIsf(p.tester, err, errorx.ErrEmptyEngine, "Expected error: %v, but got: %v",
		errorx.ErrEmptyEngine, err)
	return
}

func (p *udpProxyServer) OnShutdown(Engine) {
	assert.Zero(p.tester, atomic.LoadInt32(&p.backendBytes)%int32(len(p.packet)),
		"backend received bytes should be a multiple of packet size")
}

func (p *udpProxyServer) OnOpen(c Conn) (out []byte, action Action) {
	p.backendServerPoolMu.Lock()
	p.backendServerPool = append(p.backendServerPool, c)
	p.backendServerPoolMu.Unlock()
	if int(atomic.AddInt32(&p.backendEstablished, 1)) == len(p.backendServers) {
		// Unlock the clients to start sending data.
		close(p.stallCh)
	}
	return nil, None
}

func (p *udpProxyServer) OnTraffic(c Conn) Action {
	if c.LocalAddr().String() == p.ListenerAddr { // it's a server connection
		// Echo back to the client.
		buf, err := c.Next(-1)
		assert.NoError(p.tester, err, "Next error")
		assert.Greaterf(p.tester, len(buf), 0, "Next should not return empty buffer")
		n, err := c.Write(buf)
		assert.NoError(p.tester, err, "Write error")
		assert.EqualValuesf(p.tester, n, len(buf), "Write should send %d bytes, but sent %d bytes", len(buf), n)

		// Send the packet to a random backend server.
		p.backendServerPoolMu.Lock()
		backendServer := p.backendServerPool[rand.Intn(len(p.backendServerPool))]
		p.backendServerPoolMu.Unlock()
		err = backendServer.AsyncWrite(p.packet, nil)
		assert.NoError(p.tester, err, "AsyncWrite error")
	} else { // it's a backend connection
		buf, err := c.Next(len(p.packet))
		assert.NoError(p.tester, err, "Next error")
		assert.EqualValuesf(p.tester, buf, p.packet, "Packet mismatch, expected: %s, got: %s", p.packet, buf)
		atomic.AddInt32(&p.backendBytes, int32(len(buf)))
	}
	return None
}

func (p *udpProxyServer) OnTick() (delay time.Duration, action Action) {
	p.initBackendPoolOnce.Do(func() {
		for i, backendServer := range p.backendServers {
			network, addr, err := parseProtoAddr(backendServer)
			assert.NoError(p.tester, err, "parseProtoAddr error")
			var address net.Addr
			switch {
			case strings.HasPrefix(network, "tcp"):
				address, err = net.ResolveTCPAddr(network, addr)
			case strings.HasPrefix(network, "udp"):
				address, err = net.ResolveUDPAddr(network, addr)
			case strings.HasPrefix(network, "unix"):
				address, err = net.ResolveUnixAddr(network, addr)
			default:
				assert.Failf(p.tester, "unsupported protocol", "unsupported protocol: %s", network)
			}
			assert.NoError(p.tester, err, "ResolveNetAddr error")

			// Test the error handling with empty context.
			_, err = p.engine.Register(context.Background())
			assert.ErrorIs(p.tester, err, errorx.ErrInvalidNetworkAddress)

			var resCh <-chan RegisteredResult
			if i%2 == 0 {
				resCh, err = p.engine.Register(NewNetAddrContext(context.Background(), address))
			} else {
				c, e := net.Dial(network, addr)
				assert.NoError(p.tester, e, "Dial error")
				resCh, err = p.engine.Register(NewNetConnContext(context.Background(), c))
			}
			assert.NoError(p.tester, err, "Register connection error")
			err = goPool.DefaultWorkerPool.Submit(func() {
				res := <-resCh
				assert.NoError(p.tester, res.Err, "Register connection error")
				assert.NotNil(p.tester, res.Conn, "Register connection nil")
			})
			assert.NoError(p.tester, err, "Submit connection error")
		}
	})
	p.startClientOnce.Do(func() {
		for i := 0; i < len(p.backendServers); i++ {
			atomic.AddInt32(&p.activeClients, 1)
			err := goPool.DefaultWorkerPool.Submit(func() {
				startClient(p.tester, p.ListenerNet, p.ListenerAddr,
					true, false, p.packetSize, p.stallCh)
				atomic.AddInt32(&p.activeClients, -1)
			})
			assert.NoErrorf(p.tester, err, "Submit backend server %s error: %v", p.ListenerAddr, err)
		}
	})

	if atomic.LoadInt32(&p.activeClients) == 0 {
		return 0, Shutdown
	}

	return time.Millisecond * 200, None
}

func TestUDPProxyServer(t *testing.T) {
	t.Run("backend-udp-proxy-server", func(t *testing.T) {
		addr := "udp://127.0.0.1:10000"
		backendServers := []string{
			"udp://127.0.0.1:10001",
			"udp://127.0.0.1:10002",
			"udp://127.0.0.1:10003",
			"udp://127.0.0.1:10004",
			"udp://127.0.0.1:10005",
			"udp://127.0.0.1:10006",
			"udp://127.0.0.1:10007",
			"udp://127.0.0.1:10008",
			"udp://127.0.0.1:10009",
			"udp://127.0.0.1:10010",
		}
		t.Run("1-loop-LT", func(t *testing.T) {
			testUDPProxyServer(t, addr, backendServers, false, false)
		})
		t.Run("1-loop-ET", func(t *testing.T) {
			testUDPProxyServer(t, addr, backendServers, false, true)
		})
		t.Run("N-loop-LT", func(t *testing.T) {
			testUDPProxyServer(t, addr, backendServers, true, false)
		})
		t.Run("N-loop-ET", func(t *testing.T) {
			testUDPProxyServer(t, addr, backendServers, true, true)
		})
	})

	t.Run("backend-tcp-proxy-server", func(t *testing.T) {
		addr := "udp://127.0.0.1:20000"
		backendServers := []string{
			"tcp://127.0.0.1:20001",
			"tcp://127.0.0.1:20002",
			"tcp://127.0.0.1:20003",
			"tcp://127.0.0.1:20004",
			"tcp://127.0.0.1:20005",
			"tcp://127.0.0.1:20006",
			"tcp://127.0.0.1:20007",
			"tcp://127.0.0.1:20008",
			"tcp://127.0.0.1:20009",
			"tcp://127.0.0.1:20010",
		}
		t.Run("1-loop-LT", func(t *testing.T) {
			testUDPProxyServer(t, addr, backendServers, false, false)
		})
		t.Run("1-loop-ET", func(t *testing.T) {
			testUDPProxyServer(t, addr, backendServers, false, true)
		})
		t.Run("N-loop-LT", func(t *testing.T) {
			testUDPProxyServer(t, addr, backendServers, true, false)
		})
		t.Run("N-loop-ET", func(t *testing.T) {
			testUDPProxyServer(t, addr, backendServers, true, true)
		})
	})
}

func testUDPProxyServer(t *testing.T, addr string, backendServers []string, multicore, et bool) {
	network, address, err := parseProtoAddr(addr)
	require.NoError(t, err, "parseProtoAddr error")

	srv := udpProxyServer{
		tester:         t,
		stallCh:        make(chan struct{}),
		ListenerNet:    network,
		ListenerAddr:   address,
		backendServers: backendServers,
		packet:         []byte("andypan"),
		packetSize:     datagramLen,
	}

	var (
		backends   errgroup.Group
		netServers []io.Closer
	)
	for _, backendServer := range backendServers {
		network, addr, err := parseProtoAddr(backendServer)
		require.NoError(t, err, "parseProtoAddr error")
		if strings.HasPrefix(network, "udp") {
			udpAddr, err := net.ResolveUDPAddr(network, addr)
			require.NoError(t, err, "ResolveUDPAddr error")
			c, err := net.ListenUDP("udp", udpAddr)
			require.NoError(t, err, "ListenUDP error")
			backends.Go(func() error {
				startUDPEchoServer(t, c)
				return nil
			})
			require.NoErrorf(t, err, "Start backend UDP server %s error: %v", backendServer, err)
			netServers = append(netServers, c)
		} else {
			ln, err := net.Listen(network, addr)
			require.NoErrorf(t, err, "Create backend server %s error: %v", network+"://"+addr, err)
			backends.Go(func() error {
				startStreamEchoServer(t, ln)
				return nil
			})
			require.NoErrorf(t, err, "Start backend stream server %s error: %v", backendServer, err)
			netServers = append(netServers, ln)
		}
	}

	err = Run(&srv, addr,
		WithLoadBalancing(LeastConnections),
		WithEdgeTriggeredIO(et),
		WithMulticore(multicore),
		WithTicker(true))
	require.NoErrorf(t, err, "Run error: %v", err)

	for _, server := range netServers {
		require.NoError(t, server.Close(), "Close backend server error")
	}

	backends.Wait() //nolint:errcheck
}
