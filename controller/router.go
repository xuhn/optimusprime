// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package controller

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xuhn/optimusprime/log"

	"github.com/revel/pathtree"
)

const (
	httpStatusCode = "404"
)

type Route struct {
	//	ModuleSource        *Module         // Module name of route
	Method           string          // e.g. GET
	Path             string          // e.g. /app/:id
	Action           string          // e.g. "Application.ShowApp", "404"
	ControllerName   string          // e.g. "Application", ""
	MethodName       string          // e.g. "ShowApp", ""
	FixedParams      []string        // e.g. "arg1","arg2","arg3" (CSV formatting)
	TreePath         string          // e.g. "/GET/app/:id"
	TypeOfController *ControllerType // The controller type (if route is not wild carded)

	routesPath string // e.g. /Users/robfig/gocode/src/myapp/conf/routes
	line       int    // e.g. 3
}

type RouteMatch struct {
	Action           string // e.g. 404
	ControllerName   string // e.g. Application
	MethodName       string // e.g. ShowApp
	FixedParams      []string
	Params           map[string][]string // e.g. {id: 123}
	TypeOfController *ControllerType     // The controller type
}

/*
type ActionPathData struct {
	Key                 string            // The unique key
	ControllerName      string            // The controller name
	MethodName          string            // The method name
	Action              string            // The action
	Route               *Route            // The route
	FixedParamsByName   map[string]string // The fixed parameters
	TypeOfController    *ControllerType   // The controller type
}
*/

var (
	notFound = &RouteMatch{Action: "404"}
)

// NewRoute prepares the route to be used in matching.
func NewRoute(method, path, action, routesPath string, line int) (r *Route) {
	r = &Route{
		Method:     strings.ToUpper(method),
		Path:       path,
		Action:     action,
		TreePath:   treePath(strings.ToUpper(method), path),
		routesPath: routesPath,
		line:       line,
	}

	// URL pattern
	if !strings.HasPrefix(r.Path, "/") {
		log.ERRORF("Absolute URL required.")
		return
	}

	// Ignore the not found status code
	if action != httpStatusCode {
		found := splitActionPath(r, r.Action, false)
		if !found {
			log.PANICF("Failed to find controller for route path action %s \n", path+"?"+r.Action)
		}
	}
	return
}

func treePath(method, path string) string {
	if method == "*" {
		method = ":METHOD"
	}
	return "/" + method + path
}

type Router struct {
	Routes []*Route
	Tree   *pathtree.Node
	Module string // The module the route is associated with
	path   string // path to the routes file
}

func (router *Router) Route(req *http.Request) (routeMatch *RouteMatch) {
	// Override method if set in header
	if method := req.Header.Get("X-HTTP-Method-Override"); method != "" && req.Method == "POST" {
		req.Method = method
	}

	leaf, expansions := router.Tree.Find(treePath(req.Method, req.URL.Path))
	if leaf == nil {
		return nil
	}

	// Create a map of the route parameters.
	var params url.Values
	if len(expansions) > 0 {
		params = make(url.Values)
		for i, v := range expansions {
			params[leaf.Wildcards[i]] = []string{v}
		}
	}
	var route *Route
	var controllerName, methodName string

	// The leaf value is now a list of possible routes to match, only a controller
	routeList := leaf.Value.([]*Route)
	var typeOfController *ControllerType

	//xflog.INFOF("Found route for path %s %#v", req.URL.Path, len(routeList))
	for index := range routeList {
		route = routeList[index]
		methodName = route.MethodName
		// Special handling for explicit 404's.
		if route.Action == httpStatusCode {
			route = nil
			break
		}
		// If wildcard match on method name use the method name from the params
		if methodName[0] == ':' {
			methodName = strings.ToLower(params[methodName[1:]][0])
		}
		typeOfController = route.TypeOfController
		break
	}

	if route == nil {
		routeMatch = notFound
	} else {

		routeMatch = &RouteMatch{
			ControllerName:   controllerName,
			MethodName:       methodName,
			Params:           params,
			FixedParams:      route.FixedParams,
			TypeOfController: typeOfController,
		}
	}

	return
}

// Refresh re-reads the routes file and re-calculates the routing table.
// Returns an error if a specified action could not be found.
func (router *Router) Refresh() (err *Error) {
	router.Routes, err = parseRoutesFile(router.path, "", true)
	if err != nil {
		return
	}
	err = router.updateTree()
	return
}

func (router *Router) updateTree() *Error {
	router.Tree = pathtree.New()
	pathMap := map[string][]*Route{}

	allPathsOrdered := []string{}
	// It is possible for some route paths to overlap
	// based on wildcard matches,
	// TODO when pathtree is fixed (made to be smart enough to not require a predefined intake order) keeping the routes in order is not necessary
	for _, route := range router.Routes {
		if _, found := pathMap[route.TreePath]; !found {
			pathMap[route.TreePath] = append(pathMap[route.TreePath], route)
			allPathsOrdered = append(allPathsOrdered, route.TreePath)
		} else {
			pathMap[route.TreePath] = append(pathMap[route.TreePath], route)
		}
	}
	for _, path := range allPathsOrdered {
		routeList := pathMap[path]
		err := router.Tree.Add(path, routeList)

		// Allow GETs to respond to HEAD requests.
		if err == nil && routeList[0].Method == "GET" {
			err = router.Tree.Add(treePath("HEAD", routeList[0].Path), routeList)
		}

		// Error adding a route to the pathtree.
		if err != nil {
			return routeError(err, path, fmt.Sprintf("%#v", routeList), routeList[0].line)
		}
	}
	return nil
}

// Returns the controller namespace and name, action and module if found from the actionPath specified
func splitActionPath(r *Route, actionPath string, useCache bool) (found bool) {
	actionPath = strings.ToLower(actionPath)
	var (
		controllerName, methodName string
		typeOfController           *ControllerType
	)
	actionSplit := strings.Split(actionPath, ".")
	if len(actionSplit) == 2 {
		controllerName, methodName = strings.ToLower(actionSplit[0]), strings.ToLower(actionSplit[1])
		if i := strings.Index(methodName, "("); i > 0 {
			methodName = methodName[:i]
		}
		if controllerName[0] != ':' {
			if typeOfController == nil {
				// Check to see if we can determine the controller from only the controller name
				// an actionPath without a moduleSource will only come from
				// Scan through the controllers
				for key, controller := range controllers {
					if key == controllerName {
						// Found controller match
						typeOfController = controller
						controllerName = typeOfController.ShortName()
						found = true
						break
					}
				}
			} else {
				found = true
			}
		}
	} else {
		log.WARNF("Invalid action path %s ", actionPath)
		found = false
	}

	// Make sure no concurrent map writes occur
	if found {
		if typeOfController == nil && controllerName[0] != ':' {
			log.WARNF("Router: No controller found for %s %#v", controllerName, controllers)
		}

		if typeOfController != nil {
			// Assign controller type to avoid looking it up based on name
			r.TypeOfController = typeOfController
		}
		r.ControllerName = controllerName
		r.MethodName = methodName
	}
	return
}

// parseRoutesFile reads the given routes file and returns the contained routes.
func parseRoutesFile(routesPath, joinedPath string, validate bool) ([]*Route, *Error) {
	contentBytes, err := ioutil.ReadFile(routesPath)
	if err != nil {
		return nil, &Error{
			Title:       "Failed to load routes file",
			Description: err.Error(),
		}
	}
	return parseRoutes(routesPath, joinedPath, string(contentBytes), validate)
}

// parseRoutes reads the content of a routes file into the routing table.
func parseRoutes(routesPath, joinedPath, content string, validate bool) ([]*Route, *Error) {
	var routes []*Route

	// For each line..
	for n, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// A single route
		method, path, action, found := parseRouteLine(line)
		if !found {
			continue
		}

		route := NewRoute(method, path, action, routesPath, n)
		routes = append(routes, route)

		if validate {
			if err := validateRoute(route); err != nil {
				return nil, routeError(err, routesPath, content, n)
			}
		}
	}

	return routes, nil
}

// validateRoute checks that every specified action exists.
func validateRoute(route *Route) error {
	// Skip 404s
	if route.Action == httpStatusCode {
		return nil
	}

	// Skip variable routes.
	if route.ControllerName[0] == ':' || route.MethodName[0] == ':' {
		return nil
	}

	// Precheck to see if controller exists
	if _, found := controllers[route.ControllerName]; !found {
		// Scan through controllers to find module
		for _, c := range controllers {
			controllerName := strings.ToLower(c.Type.Name())
			if controllerName == route.ControllerName {
				log.WARNF("Matched empty namespace route for %s for the route %s", controllerName, route.Path)
			}
		}
	}

	// TODO need to check later
	// does it do only validation or validation and instantiate the controller.
	var c Controller
	return c.SetTypeAction(route.ControllerName, route.MethodName, route.TypeOfController)
}

// routeError adds context to a simple error message.
func routeError(err error, routesPath, content string, n int) *Error {
	if revelError, ok := err.(*Error); ok {
		return revelError
	}
	// Load the route file content if necessary
	if content == "" {
		if contentBytes, er := ioutil.ReadFile(routesPath); er != nil {
			log.ERRORF("Failed to read route file %s: %s\n", routesPath, er)
		} else {
			content = string(contentBytes)
		}
	}
	return &Error{
		Title:       "Route validation error",
		Description: err.Error(),
		Path:        routesPath,
		Line:        n + 1,
		SourceLines: strings.Split(content, "\n"),
	}
}

// Groups:
// 1: method
// 4: path
// 5: action
// 6: fixedargs
var routePattern = regexp.MustCompile(
	"(?i)^(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD|WS|\\*)" +
		"[(]?([^)]*)(\\))?[ \t]+" +
		"(.*/[^ \t]*)[ \t]+([^ \t(]+)" +
		`\(?([^)]*)\)?[ \t]*$`)

func parseRouteLine(line string) (method, path, action string, found bool) {
	matches := routePattern.FindStringSubmatch(line)
	if matches == nil {
		return
	}
	method, path, action = matches[1], matches[4], matches[5]
	found = true
	return
}

func NewRouter(routesPath string) *Router {
	return &Router{
		Tree: pathtree.New(),
		path: routesPath,
	}
}

type ActionDefinition struct {
	Host, Method, URL, Action string
	Star                      bool
	Args                      map[string]string
}

func (a *ActionDefinition) String() string {
	return a.URL
}

/*
func (router *Router) Reverse(action string, argValues map[string]string) (ad *ActionDefinition) {
	pathData, found := splitActionPath(nil, action, true)

	if found {
		if pathData.Route == nil {
			var possibleRoute *Route
			// If the route is nil then we need to go through the routes to find the first matching route
			// from this controllers namespace, this is likely a wildcard route match
			for _, route := range router.Routes {
				// Skip routes that are not wild card or empty
				if route.ControllerName == "" || route.MethodName == "" {
					continue
				}
				if route.ModuleSource == pathData.ModuleSource && route.ControllerName[0] == ':' {
					// Wildcard match in same module space
					pathData.Route = route
					break
				} else if route.ActionPath() == pathData.ModuleSource.Namespace()+pathData.ControllerName {
					// Action path match
					pathData.Route = route
					break
				} else if route.ControllerName == pathData.ControllerName {
					// Controller name match
					possibleRoute = route
				}
			}
			if pathData.Route == nil && possibleRoute != nil {
				pathData.Route = possibleRoute
				xflog.WARNF("For reverse action %s matched path route %#v", action, possibleRoute)
			}
			if pathData.Route != nil {
				TRACE.Printf("Reverse Storing recognized action path %s for route %#v\n", action, pathData.Route)
			}
		}

		// Likely unknown route because of a wildcard, perform manual lookup
		if pathData.Route != nil {
			route := pathData.Route

			// If the controller or method are wildcards we need to populate the argValues
			controllerWildcard := route.ControllerName[0] == ':'
			methodWildcard := route.MethodName[0] == ':'

			// populate route arguments with the names
			if controllerWildcard {
				argValues[route.ControllerName[1:]] = pathData.ControllerName
			}
			if methodWildcard {
				argValues[route.MethodName[1:]] = pathData.MethodName
			}
			// In theory all routes should be defined and pre-populated, the route controllers may not be though
			// with wildcard routes
			if pathData.TypeOfController == nil {
				if controllerWildcard || methodWildcard {
					if controller := ControllerTypeByName(pathData.ControllerNamespace+pathData.ControllerName, route.ModuleSource); controller != nil {
						// Wildcard match boundary
						pathData.TypeOfController = controller
						// See if the path exists in the module based
					} else {
						xflog.ERORRF("Controller %s not found in reverse lookup", pathData.ControllerNamespace+pathData.ControllerName)
						return
					}
				}
			}

			if pathData.TypeOfController == nil {
				xflog.ERORRF("Controller %s not found in reverse lookup", pathData.ControllerNamespace+pathData.ControllerName)
				return
			}
			var (
				queryValues  = make(url.Values)
				pathElements = strings.Split(route.Path, "/")
			)
			for i, el := range pathElements {
				if el == "" || (el[0] != ':' && el[0] != '*') {
					continue
				}

				val, ok := argValues[el[1:]]
				if !ok {
					val = "<nil>"
					xflog.ERORRF("revel/router: reverse route missing route arg ", el[1:])
				}
				pathElements[i] = val
				delete(argValues, el[1:])
				continue
			}

			// Add any args that were not inserted into the path into the query string.
			for k, v := range argValues {
				queryValues.Set(k, v)
			}

			// Calculate the final URL and Method
			url := strings.Join(pathElements, "/")
			if len(queryValues) > 0 {
				url += "?" + queryValues.Encode()
			}

			method := route.Method
			star := false
			if route.Method == "*" {
				method = "GET"
				star = true
			}

			//xflog.INFOF("Reversing action %s to %s Using Route %#v",action,url,pathData.Route)

			return &ActionDefinition{
				URL:    url,
				Method: method,
				Star:   star,
				Action: action,
				Args:   argValues,
				Host:   "TODO",
			}
		}
	}
	xflog.ERORRF("Failed to find reverse route:", action, argValues)
	return nil
}
*/
func RouterFilter(c *Controller, fc []Filter) {
	// Figure out the Controller/Action
	route := MainRouter.Route(c.Request.Request)
	if route == nil {
		c.Result = c.NotFound("No matching route found: " + c.Request.RequestURI)
		return
	}

	// The route may want to explicitly return a 404.
	if route.Action == httpStatusCode {
		c.Result = c.NotFound("(intentionally)")
		return
	}

	// Set the action.
	if err := c.SetTypeAction(route.ControllerName, route.MethodName, route.TypeOfController); err != nil {
		c.Result = c.NotFound(err.Error())
		return
	}

	// Add the route and fixed params to the Request Params.
	c.Params.Route = route.Params

	// Add the fixed parameters mapped by name.
	// TODO: Pre-calculate this mapping.
	for i, value := range route.FixedParams {
		if c.Params.Fixed == nil {
			c.Params.Fixed = make(url.Values)
		}
		if i < len(c.MethodType.Args) {
			arg := c.MethodType.Args[i]
			c.Params.Fixed.Set(arg.Name, value)
		} else {
			log.WARNF("Too many parameters to", route.Action, "trying to add", value)
			break
		}
	}

	fc[0](c, fc[1:])
}

// HTTPMethodOverride overrides allowed http methods via form or browser param
func HTTPMethodOverride(c *Controller, fc []Filter) {
	// An array of HTTP verbs allowed.
	verbs := []string{"POST", "PUT", "PATCH", "DELETE"}

	method := strings.ToUpper(c.Request.Request.Method)

	if method == "POST" {
		param := strings.ToUpper(c.Request.Request.PostFormValue("_method"))

		if len(param) > 0 {
			override := false
			// Check if param is allowed
			for _, verb := range verbs {
				if verb == param {
					override = true
					break
				}
			}

			if override {
				c.Request.Request.Method = param
			} else {
				c.Response.Status = 405
				c.Result = c.RenderError(&Error{
					Title:       "Method not allowed",
					Description: "Method " + param + " is not allowed (valid: " + strings.Join(verbs, ", ") + ")",
				})
				return
			}

		}
	}

	fc[0](c, fc[1:]) // Execute the next filter stage.
}

func init() {
	OnAppStart(func() {
		MainRouter = NewRouter(filepath.Join(BasePath, "conf", "routes"))
		err := MainRouter.Refresh()
		if err != nil {
			// Not in dev mode and Route loading failed, we should crash.
			log.PANICF(err.Error())
		}
	})
}
