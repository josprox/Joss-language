package core

import (
	"database/sql"
	"encoding/json"

	"github.com/jossecurity/joss/pkg/parser"
)

// NativeHandler is a function that executes a native method
// NativeHandler is a function that executes a native method
type NativeHandler func(r *Runtime, instance *Instance, method string, args []interface{}) interface{}

// SEOData stores metadata for search engine optimization
type SEOData struct {
	Title       string
	Description string
	Keywords    []string
	Canonical   string
	Meta        map[string]string
	OG          map[string]string
}

// SitemapEntry represents a URL in the sitemap
type SitemapEntry struct {
	URL        string
	LastMod    string
	ChangeFreq string
	Priority   float64
}

// Runtime manages the execution environment of a Joss program
type Runtime struct {
	Env               map[string]string
	Variables         map[string]interface{}
	VarTypes          map[string]string // For strict typing
	Classes           map[string]*parser.ClassStatement
	Functions         map[string]*parser.MethodStatement
	DB                *sql.DB
	Routes            map[string]map[string]interface{} // HTTP Method -> Path -> Handler
	CurrentMiddleware []string
	CustomMiddlewares map[string]interface{} // Name -> Closure/Handler
	NativeHandlers    map[string]NativeHandler

	// SEO & Sitemap
	SEO            *SEOData
	SitemapEntries []SitemapEntry
	CurrentSource  string // "routes", "api", "app", etc.
}

// Instance represents an instance of a class
type Instance struct {
	Class  *parser.ClassStatement
	Fields map[string]interface{}
}

func (i *Instance) MarshalJSON() ([]byte, error) {
	if i == nil {
		return []byte("null"), nil
	}
	return json.Marshal(i.Fields)
}

// BoundMethod represents a method bound to an instance
type BoundMethod struct {
	Method      *parser.MethodStatement
	Instance    *Instance
	StaticClass string // For static calls
}

// Future represents an asynchronous computation
type Future struct {
	done   chan bool
	result interface{}
	err    error
}

// Channel represents a Go channel
type Channel struct {
	Ch chan interface{}
}

func (c *Channel) String() string { return "channel" }

// ReturnPanic is used to bubble up ReturnStatements through the AST
type ReturnPanic struct {
	Value interface{}
}

// BreakPanic is used to exit a loop
type BreakPanic struct{}

// ContinuePanic is used to skip to the next loop iteration
type ContinuePanic struct{}

// Wait blocks until the Future completes and returns the result
func (f *Future) Wait() interface{} {
	<-f.done
	if f.err != nil {
		panic(f.err)
	}
	return f.result
}

// Cout represents standard output stream
type Cout struct{}

func (c *Cout) String() string { return "cout" }

// Cin represents standard input stream
type Cin struct{}

func (c *Cin) String() string { return "cin" }
