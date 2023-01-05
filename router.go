package sandwich

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/augustoroman/sandwich/chain"
)

// Router implements the sandwich middleware chaining and routing functionality.
type Router interface {
	// Set a value that will be available to all handlers subsequent referenced.
	// This is typically used for concrete values. For interfaces to be correctly
	// provided to subsequent middleware, use SetAs.
	Set(vals ...any)
	// SetAs sets a value as the specified interface that will be available to all
	// handlers.
	//
	// Example:
	//    type DB interface { ... }
	//    var db DB = ...
	//    mux.SetAs(db, (*DB)(nil))
	//
	// That is functionally equivalent to using a middleware function that returns
	// the desired interface instance:
	//    type DB interface { ... }
	//    var db DB = ...
	//    mux.Use(func() DB { return db })
	SetAs(val, ifacePtr any)

	// Use adds middleware to be invoked for all routes registered by the
	// returned Router. The current router is not affected. This is equivalent to
	// adding the specified middelwareHandlers to each registered route.
	Use(middlewareHandlers ...any)

	// On will register a handler for the given method and path.
	On(method, path string, handlers ...any)

	// Get registers handlers for the specified path for the 'GET' HTTP method.
	// Get is shorthand for `On("GET", ...)`.
	Get(path string, handlers ...any)
	// Put registers handlers for the specified path for the 'PUT' HTTP method.
	// Put is shorthand for `On("PUT", ...)`.
	Put(path string, handlers ...any)
	// Post registers handlers for the specified path for the 'POST' HTTP method.
	// Post is shorthand for `On("POST", ...)`.
	Post(path string, handlers ...any)
	// Patch registers handlers for the specified path for the 'PATCH' HTTP
	// method. Patch is shorthand for `On("PATCH", ...)`.
	Patch(path string, handlers ...any)
	// Delete registers handlers for the specified path for the 'DELETE' HTTP
	// method. Delete is shorthand for `On("DELETE", ...)`.
	Delete(path string, handlers ...any)
	// Any registers a handlers for the specified path for any HTTP method. This
	// will always be superceded by dedicated method handlers. For example, if the
	// path '/users/:id/' is registered for Get, Put and Any, GET and PUT requests
	// will be handled by the Get(...) and Put(...) registrations, but DELETE,
	// CONNECT, or HEAD would be handled by the Any(...) registration. Any is a
	// shortcut for `On("*", ...)`.
	Any(path string, handlers ...any)

	// OnErr uses the specified error handler to handle any errors that occur on
	// any routes in this router.
	OnErr(handler any)

	// SubRouter derives a router that will called for all suffixes (and methods)
	// for the specified path. For example, `sub := root.SubRouter("/api")` will
	// create a router that will handle `/api/`, `/api/foo`.
	SubRouter(pathPrefix string) Router

	// ServeHTTP implements the http.Handler interface for the router.
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// BuildYourOwn returns a minimal router that has no initial middleware
// handling.
func BuildYourOwn() Router {
	r := &router{}
	r.base = r.base.Arg((*http.ResponseWriter)(nil))
	r.base = r.base.Arg((*http.Request)(nil))
	r.base = r.base.Arg((Params)(nil))
	return r
}

// TheUsual returns a router initialized with useful middleware.
func TheUsual() Router {
	r := BuildYourOwn()
	r.Use(WrapResponseWriter, LogRequests)
	r.OnErr(HandleError)
	return r
}

type router struct {
	base       chain.Func
	subRouters map[string]*router
	byMethod   map[string]*mux
	anyMethod  *mux
	notFound   http.Handler
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	params := Params{}
	h := r.match(req.Method, req.URL.Path, params)
	if h != nil {
		h.ServeHTTP(w, req, params)
	} else if r.notFound != nil {
		r.notFound.ServeHTTP(w, req)
	} else {
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (r *router) SubRouter(prefix string) Router {
	if r.subRouters == nil {
		r.subRouters = map[string]*router{}
	}
	prefix = strings.TrimRight(prefix, "/") + "/"
	for existingPrefix := range r.subRouters {
		if existingPrefix == prefix || strings.HasPrefix(existingPrefix, prefix) || strings.HasPrefix(prefix, existingPrefix) {
			panic(fmt.Sprintf(
				"SubRouter with prefix %#q conflicts with existing SubRouter with prefix %#q",
				prefix, existingPrefix,
			))
		}
	}
	r.subRouters[prefix] = &router{
		base:     r.base,
		notFound: r.notFound,
	}
	return r.subRouters[prefix]
}

func (r *router) match(method, uri string, params Params) httpHandlerWithParams {
	method = strings.ToUpper(method)
	for prefix, sub := range r.subRouters {
		if strings.HasPrefix(uri, prefix) {
			return sub.match(method, strings.TrimPrefix(uri, prefix), params)
		}
	}
	if h := r.byMethod[method].Match(uri, params); h != nil {
		return h
	}
	if h := r.anyMethod.Match(uri, params); h != nil {
		return h
	}
	return nil
}

func (r *router) Set(vals ...any) {
	for _, val := range vals {
		r.base = r.base.Set(val)
	}
}

func (r *router) SetAs(val, ifacePtr any) {
	r.base = r.base.SetAs(val, ifacePtr)
}

func (r *router) Use(middlewareHandlers ...any) {
	r.base = apply(r.base, middlewareHandlers...)
}

func (r *router) OnErr(errorHandler any) {
	r.base = r.base.OnErr(errorHandler)
}

func (r *router) On(method, path string, handlers ...any) {
	method = strings.ToUpper(method)
	m := r.getOrAllocateMux(method)
	if err := m.Register(path, handler{apply(r.base, handlers...)}); err != nil {
		panic(fmt.Errorf("Cannot register route: %v", err))
	}
}

func (r *router) Any(path string, handlers ...any)    { r.On("*", path, handlers...) }
func (r *router) Get(path string, handlers ...any)    { r.On("GET", path, handlers...) }
func (r *router) Put(path string, handlers ...any)    { r.On("PUT", path, handlers...) }
func (r *router) Post(path string, handlers ...any)   { r.On("POST", path, handlers...) }
func (r *router) Patch(path string, handlers ...any)  { r.On("PATCH", path, handlers...) }
func (r *router) Delete(path string, handlers ...any) { r.On("DELETE", path, handlers...) }

func (r *router) getOrAllocateMux(method string) *mux {
	if method == "*" {
		if r.anyMethod == nil {
			r.anyMethod = &mux{}
		}
		return r.anyMethod
	}
	if r.byMethod == nil {
		r.byMethod = map[string]*mux{}
	}
	m := r.byMethod[method]
	if m == nil {
		m = &mux{}
		r.byMethod[method] = m
	}
	return m
}

type handler struct{ chain.Func }

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request, p Params) {
	h.Func.MustRun(w, r, p)
}

type Params map[string]string

type mux struct {
	static  map[string]*mux
	params  []muxParam
	handler httpHandlerWithParams
}

type muxParam struct {
	paramName string
	greedy    bool
	mux       *mux
}

type httpHandlerWithParams interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, p Params)
}

func (m *mux) Register(pattern string, h httpHandlerWithParams) error {
	if !strings.HasPrefix(pattern, "/") {
		return errors.New("patterns must begin with /")
	}
	segments := strings.Split(pattern[1:], "/")
	reg := registerInfo{
		seenParams: map[string]bool{},
		seenGreedy: false,
	}
	if m.static == nil {
		m.static = map[string]*mux{}
	}
	if err := reg.registerSegments(m, segments, h); err != nil {
		return fmt.Errorf("%#q: bad pattern: %w", pattern, err)
	}
	return nil
}

type registerInfo struct {
	seenParams map[string]bool
	seenGreedy bool
}

func (r *registerInfo) registerSegments(m *mux, segments []string, h httpHandlerWithParams) error {
	if len(segments) == 0 {
		if m.handler != nil {
			return fmt.Errorf("repeated entry")
		}
		m.handler = h
		return nil
	}
	next, remaining := segments[0], segments[1:]
	if strings.HasPrefix(next, "::") {
		return r.registerStatic(m, next[1:], remaining, h)
	} else if strings.HasPrefix(next, ":") {
		return r.registerParam(m, next[1:], remaining, h)
	} else {
		return r.registerStatic(m, next, remaining, h)
	}
}

func (r *registerInfo) registerStatic(m *mux, path string, remaining []string, h httpHandlerWithParams) error {
	sub := m.static[path]
	if sub == nil {
		sub = &mux{
			static: map[string]*mux{},
		}
	}
	err := r.registerSegments(sub, remaining, h)
	if err == nil {
		m.static[path] = sub
	}
	return err
}

func (r *registerInfo) registerParam(m *mux, param string, remaining []string, h httpHandlerWithParams) error {
	greedy := strings.HasSuffix(param, "*")
	name := strings.TrimSuffix(param, "*")
	if greedy && r.seenGreedy {
		return fmt.Errorf("only one greedy param allowed per pattern: %#q", name)
	} else if r.seenParams[name] {
		return fmt.Errorf("param used twice: %#q", name)
	}
	// Check to see if the param already exists. E.g. we've already registered
	// param at this level via:
	//    /root/:param/path1 --> h1
	// and now we're registering:
	//    /root/:param/path2 --> h2
	for _, p := range m.params {
		if p.paramName == name {
			if p.greedy != greedy {
				return fmt.Errorf("param %#q is sometimes greedy and sometimes not", name)
			}
			return r.registerSegments(p.mux, remaining, h)
		}
		// If we haven't registered this one yet, then we need to avoid ambiguous
		// path registrations. For example:
		//   /root/:p1/path
		//   /root/:p2/path
		// should not be allowed, nor should:
		//   /root/:p1/:x/:y
		//   /root/:p2/:a/:b
		if err := p.mux.checkAmbiguous(remaining); err != nil {
			return fmt.Errorf("ambiguous route: %w", err)
		}
	}
	sub := &mux{
		static: map[string]*mux{},
	}
	r.seenParams[name] = true
	r.seenGreedy = r.seenGreedy || greedy
	err := r.registerSegments(sub, remaining, h)
	if err == nil {
		m.params = append(m.params, muxParam{
			paramName: name,
			greedy:    greedy,
			mux:       sub,
		})
	}
	return err
}

func (m *mux) checkAmbiguous(segments []string) error {
	if len(segments) == 0 {
		if m.handler != nil {
			return fmt.Errorf("ambiguous route")
		}
		return nil
	}
	static, isStatic, _, _ := entryToInfo(segments[0])
	if isStatic {
		if child := m.static[static]; child != nil {
			return child.checkAmbiguous(segments[1:])
		}
		return nil
	}
	for _, p := range m.params {
		if err := p.mux.checkAmbiguous(segments[1:]); err != nil {
			return err
		}
	}
	return nil
}

func entryToInfo(entry string) (static string, isStatic bool, paramName string, greedy bool) {
	if strings.HasPrefix(entry, "::") {
		// double colon prefix escapes to single colon static path name.
		return entry[1:], true, "", false
	} else if !strings.HasPrefix(entry, ":") {
		return entry, true, "", false
	}
	paramName = strings.TrimSuffix(entry[1:], "*")
	greedy = strings.HasSuffix(entry, "*")
	return "", false, paramName, greedy
}

func (m *mux) Match(uri string, params Params) httpHandlerWithParams {
	uri = strings.TrimPrefix(uri, "/")
	segments := strings.Split(uri, "/")
	matched := m.matchPrefix(segments, params)
	if matched == nil {
		return nil
	}
	return matched
}

func (m *mux) matchPrefix(segments []string, params Params) httpHandlerWithParams {
	if m == nil {
		return nil
	}
	if len(segments) == 0 {
		return m.handler
	}
	path, remaining := segments[0], segments[1:]
	if sub := m.static[path]; sub != nil {
		match := sub.matchPrefix(remaining, params)
		if match != nil {
			return match
		}
	}
	for _, param := range m.params {
		if !param.greedy {
			matched := param.mux.matchPrefix(remaining, params)
			if matched != nil {
				params[param.paramName] = path
				return matched
			}
		} else {
			matched, used := param.mux.matchSuffix(remaining, params)
			if matched != nil {
				N := len(segments)
				params[param.paramName] = strings.Join(segments[:N-used], "/")
				return matched
			}
		}
	}
	return nil
}

func (m *mux) matchSuffix(segments []string, params Params) (h httpHandlerWithParams, depth int) {
	N := len(segments)
	if N == 0 {
		return m.handler, 0
	}
	for staticPath, sub := range m.static {
		match, d := sub.matchSuffix(segments, params)
		if match == nil {
			continue
		}
		depth = d + 1
		actualPath := segments[N-depth]
		if actualPath != staticPath {
			continue
		}
		return match, depth
	}
	for _, param := range m.params {
		match, d := param.mux.matchSuffix(segments, params)
		if match == nil {
			continue
		}
		depth = d + 1
		actualPath := segments[N-depth]
		params[param.paramName] = actualPath // TODO: might be rejected, might spam params
		return match, depth
	}
	return m.handler, 0
}
