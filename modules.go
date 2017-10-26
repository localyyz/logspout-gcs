package main

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	_ "github.com/gliderlabs/logspout/adapters/syslog"
	"github.com/gliderlabs/logspout/router"

	_ "github.com/gliderlabs/logspout/transports/tcp"
	_ "github.com/gliderlabs/logspout/transports/tls"
	_ "github.com/gliderlabs/logspout/transports/udp"

	"github.com/oklog/ulid"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

const MB = 1 << 20

var entropy io.Reader

func init() {
	t := time.Now()
	entropy = rand.New(rand.NewSource(t.UnixNano()))

	router.AdapterFactories.Register(NewGCSAdapter, "gcs")
}

func NewGCSAdapter(route *router.Route) (router.LogAdapter, error) {
	// Default 2 minute log flushing interval to push logs to S3
	flushInterval := time.Duration(2 * time.Minute)
	if s := os.Getenv("FLUSH_INTERVAL"); s != "" {
		i, _ := strconv.ParseInt(s, 10, 64)
		if i > 0 {
			flushInterval = time.Duration(time.Duration(i) * time.Second)
		}
	}

	// Default 16MB log buffer before flushing
	maxSinkSizeMB := 16
	if s := os.Getenv("MAX_SINK_SIZE_MB"); s != "" {
		i, _ := strconv.Atoi(s)
		if i > 0 {
			maxSinkSizeMB = i
		}
	}

	// Parse GCS bucket and storage path from route address
	bucketID := route.Address
	storePath, _ := route.Options["path"]

	var opts []option.ClientOption
	// Find and add key file to client options
	if keyFile := os.Getenv("GCS_KEY_FILE"); keyFile != "" {
		opts = append(opts, option.WithServiceAccountFile(keyFile))
	}

	ctx := context.Background()
	conn, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Verify connection works by check/writing pong to the bucket.
	ping := conn.Bucket(bucketID).Object("ping")
	attr, err := ping.Attrs(ctx)
	if err != nil && err != storage.ErrObjectNotExist {
		return nil, err
	}
	if attr == nil {
		wc := ping.NewWriter(ctx)
		if _, err := fmt.Fprintf(wc, "ping!"); err != nil {
			return nil, err
		}
		if err := wc.Close(); err != nil {
			return nil, err
		}
	}

	a := &GCSAdapter{
		route:     route,
		bucketID:  bucketID,
		storePath: storePath,
		conn:      conn,

		recordCh:      make(chan logEntry),
		logSink:       newLogSink(),
		maxSinkSize:   int64(maxSinkSizeMB * MB),
		flushInterval: flushInterval,
	}
	return a, nil
}

type GCSAdapter struct {
	route     *router.Route
	bucketID  string
	storePath string
	conn      *storage.Client

	recordCh      chan logEntry
	logSink       *logSink
	maxSinkSize   int64 // in bytes
	flushInterval time.Duration
}

type logEntry struct {
	Container string
	Message   string
}

// Stream sends log data to a connection
func (a *GCSAdapter) Stream(logstream chan *router.Message) {
	go a.recorder(a.flushInterval)

	for message := range logstream {
		a.recordCh <- logEntry{
			Container: message.Container.Name,
			Message:   message.Data,
		}
	}
}

func (a *GCSAdapter) recorder(d time.Duration) {
	ticker := time.NewTicker(d)
	for {
		select {

		case entry := <-a.recordCh:
			a.logSink.Push(entry)

			if a.logSink.numBytes >= a.maxSinkSize {
				go a.sendLogs()
			}

		case <-ticker.C:
			// TODO: we could check error response, after X consequtive errors
			// we can stop processing, or at least stop for some period of time.
			go a.sendLogs()

		}
	}
}

func (a *GCSAdapter) sendLogs() error {
	// Check if any entries to send and preempt if not
	a.logSink.Lock()
	if len(a.logSink.entries) == 0 {
		a.logSink.Unlock()
		return nil
	}

	// Create container index from sink entries
	logs := map[string][]string{}
	for _, e := range a.logSink.entries {
		logs[e.Container] = append(logs[e.Container], e.Message)
	}

	a.logSink.Reset()
	a.logSink.Unlock()

	// computed key name becomes: /{storePath}/{containerName}/{unix-ts}-{ulid}.log
	t := time.Now().Unix()
	ulid := newULID()
	filename := fmt.Sprintf("%d-%s", t, ulid.String())

	for name, msgs := range logs {
		if len(msgs) == 0 {
			continue
		}

		key := fmt.Sprintf("%s%s/%s.log", a.storePath, name, filename)
		data := []byte(strings.Join(msgs, "\r\n"))

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, time.Duration(5*time.Minute))
		defer cancel()

		objectWriter := a.conn.Bucket(a.bucketID).Object(key).NewWriter(ctx)
		if _, err := objectWriter.Write(data); err != nil {
			return err
		}

		if err := objectWriter.Close(); err != nil {
			return err
		}
	}

	return nil
}

type logSink struct {
	sync.Mutex
	entries  []logEntry
	numBytes int64
}

func newLogSink() *logSink {
	return &logSink{
		entries: []logEntry{},
	}
}

func (s *logSink) Push(e logEntry) {
	s.Lock()
	s.numBytes += int64(len(e.Message))
	s.entries = append(s.entries, e)
	s.Unlock()
}

func (s *logSink) Reset() {
	s.entries = s.entries[:0]
	s.numBytes = 0
}

func newULID() ulid.ULID {
	t := time.Now()
	return ulid.MustNew(ulid.Timestamp(t), entropy)
}
