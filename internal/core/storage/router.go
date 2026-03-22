package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Router holds optional storage backends. SQLite uses the already-vendored modernc driver.
// MySQL / Mongo / Redis / Kafka keep DSN or broker strings for orchestration; full protocol
// clients can be added via `go get` + thin wrappers (see docs/en/CORE_SERVICES.md).
type Router struct {
	// SQLiteCore optional second DB for plugin key-value / dedup (separate from app data.db).
	SQLiteCore *sql.DB

	MySQLDSN    string
	MongoURI    string
	MongoDBName string
	RedisAddr   string
	KafkaBrokers []string
}

// NewRouterFromEnv configures from:
//   - TRACKER_CORE_SQLITE — opens sqlite DB for core plugin storage
//   - TRACKER_MYSQL_DSN — recorded for drivers / external tools
//   - TRACKER_MONGO_URI, TRACKER_MONGO_DATABASE
//   - TRACKER_REDIS_ADDR
//   - TRACKER_KAFKA_BROKERS (comma-separated)
func NewRouterFromEnv() *Router {
	r := &Router{
		MySQLDSN:      strings.TrimSpace(os.Getenv("TRACKER_MYSQL_DSN")),
		MongoURI:      strings.TrimSpace(os.Getenv("TRACKER_MONGO_URI")),
		MongoDBName:   strings.TrimSpace(os.Getenv("TRACKER_MONGO_DATABASE")),
		RedisAddr:     strings.TrimSpace(os.Getenv("TRACKER_REDIS_ADDR")),
	}
	if r.MongoDBName == "" {
		r.MongoDBName = "tracker"
	}
	if brokers := strings.TrimSpace(os.Getenv("TRACKER_KAFKA_BROKERS")); brokers != "" {
		for _, b := range strings.Split(brokers, ",") {
			b = strings.TrimSpace(b)
			if b != "" {
				r.KafkaBrokers = append(r.KafkaBrokers, b)
			}
		}
	}
	if path := strings.TrimSpace(os.Getenv("TRACKER_CORE_SQLITE")); path != "" {
		db, err := sql.Open("sqlite", path)
		if err == nil {
			db.SetMaxOpenConns(8)
			r.SQLiteCore = db
		}
	}
	return r
}

// Close closes opened pools.
func (r *Router) Close() error {
	if r == nil || r.SQLiteCore == nil {
		return nil
	}
	return r.SQLiteCore.Close()
}

// Ping runs best-effort checks: SQLite ping, TCP dial for redis/mongo/mysql/kafka endpoints.
func (r *Router) Ping(ctx context.Context) map[string]error {
	out := make(map[string]error)
	if r == nil {
		return out
	}
	if r.SQLiteCore != nil {
		out["sqlite_core"] = r.SQLiteCore.PingContext(ctx)
	}
	if r.MySQLDSN != "" {
		addr := dsnHostPort(r.MySQLDSN, "3306")
		out["mysql_tcp"] = dialTCP(ctx, addr)
	}
	if r.MongoURI != "" {
		addr := mongoHostPort(r.MongoURI)
		if addr != "" {
			out["mongo_tcp"] = dialTCP(ctx, addr)
		}
	}
	if r.RedisAddr != "" {
		out["redis_tcp"] = dialTCP(ctx, r.RedisAddr)
	}
	if len(r.KafkaBrokers) > 0 {
		out["kafka_tcp"] = dialTCP(ctx, r.KafkaBrokers[0])
	}
	return out
}

func dialTCP(ctx context.Context, address string) error {
	if address == "" {
		return fmt.Errorf("empty address")
	}
	d := net.Dialer{Timeout: 3 * time.Second}
	c, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	return c.Close()
}

// dsnHostPort extracts host for TCP check from "user:pass@tcp(host:port)/db" or "host:port".
func dsnHostPort(dsn, defaultPort string) string {
	// user:pass@tcp(127.0.0.1:3306)/dbname
	if i := strings.Index(dsn, "tcp("); i >= 0 {
		rest := dsn[i+4:]
		if j := strings.Index(rest, ")"); j > 0 {
			return rest[:j]
		}
	}
	if strings.Contains(dsn, ":") && !strings.Contains(dsn, "@") {
		return dsn
	}
	return ""
}

func mongoHostPort(uri string) string {
	// mongodb://host:27017/db
	u := strings.TrimPrefix(uri, "mongodb://")
	u = strings.TrimPrefix(u, "mongodb+srv://")
	if idx := strings.Index(u, "/"); idx > 0 {
		u = u[:idx]
	}
	if idx := strings.Index(u, "?"); idx > 0 {
		u = u[:idx]
	}
	if u == "" {
		return ""
	}
	if !strings.Contains(u, ":") {
		return u + ":27017"
	}
	return u
}
