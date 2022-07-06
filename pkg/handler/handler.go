package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/koblas/swerver/pkg/minimatch"
	pathToRegExp "github.com/koblas/swerver/pkg/path_to_regexp"
)

type HandlerState struct {
	Configuration
	logger Logger
}

// Implements http.Handler
func NewHandler(config Configuration) HandlerState {
	state := HandlerState{
		Configuration: config,
		logger:        NewLogger(config.Debug),
	}

	// return gziphandler.GzipHandler(state)
	return state
}

func acceptJSON(r *http.Request) bool {
	accept := r.Header[http.CanonicalHeaderKey("accept")]

	for _, value := range accept {
		if strings.Contains(strings.ToLower(value), "application/json") {
			return true
		}
	}

	return false
}

func (state HandlerState) serveFile(w http.ResponseWriter, r *http.Request, name string) {
	f, err := os.Open(name)
	if err != nil {
		//msg, code := toHTTPError(err)
		//Error(w, msg, code)
		return
	}
	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		//msg, code := toHTTPError(err)
		//Error(w, msg, code)
		return
	}

	http.ServeContent(w, r, d.Name(), d.ModTime(), f)
}

func (state HandlerState) sendError(w http.ResponseWriter, r *http.Request, path string, statusCode int) {
	errorPage := filepath.Join(state.Public, path, fmt.Sprintf("%d.html", statusCode))
	_, err := os.Lstat(errorPage)
	if err == nil {
		w.WriteHeader(statusCode)
		state.serveFile(w, r, errorPage)
		return
	}

	type errorBodyType = struct {
		StatusCode int    `json:"-"`
		Code       string `json:"code"`
		Message    string `json:"message"`
	}
	type errorInfo = struct {
		Error errorBodyType `json:"error"`
	}

	errorBody := errorBodyType{StatusCode: statusCode}
	switch statusCode {
	case http.StatusBadRequest:
		errorBody.Code = "bad_request"
		errorBody.Message = "Bad request"
	case http.StatusNotFound:
		errorBody.Code = "not_found"
		errorBody.Message = "The requested path could not be found"
	case http.StatusInternalServerError:
		errorBody.Code = "internal_server_error"
		errorBody.Message = "A server error has occurred"
	}

	if acceptJSON(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(errorInfo{errorBody}); err != nil {
			log.Fatal(err)
		}

		return
	}

	w.WriteHeader(statusCode)

	err = errorTemplate.Execute(w, errorBody)

	if err != nil {
		log.Fatal(err)
	}
}

func slasher(value string) string {
	normalize := func(value string) string {
		return path.Join("/", value)
	}

	if len(value) > 0 && value[0] == '!' {
		return "!" + normalize(value[1:])
	}

	return normalize(value)
}

func sourceMatches(source string, requestPath string, allowSegments bool) (bool, []pathToRegExp.Token, []string) {
	keys := []pathToRegExp.Token{}
	slashed := slasher(source)
	resolvedPath := path.Clean(requestPath)

	if allowSegments {
		normalized := strings.Replace(slashed, "*", "(.*)", -1)
		matcher, err := pathToRegExp.PathToRegexp(normalized, pathToRegExp.NewOptions())
		if err != nil {
			return false, keys, []string{}
		}

		didMatch, result := matcher.MatchString(resolvedPath)

		if didMatch {
			return true, keys, result.Results
		}
	}

	if ok, _ := minimatch.MatchString(resolvedPath, slashed, minimatch.Options{}); ok {
		return true, keys, []string{}
	}

	return false, keys, []string{}
}

func applyRewrites(path string, rewrites []ConfigRewrite, repetitive bool) *string {
	var fallback *string

	if len(rewrites) == 0 {
		return &path
	}

	rewritesCopy := rewrites[:]
	offset := 0
	for idx, item := range rewrites {
		target := toTarget(item.Source, item.Destination, path)

		if target != nil {
			// Remove rules that were already applied
			copy(rewritesCopy[:idx-offset], rewritesCopy[:idx-offset+1])
			rewritesCopy = rewritesCopy[:len(rewritesCopy)-1]
			offset++

			return applyRewrites(slasher(*target), rewritesCopy, true)
		}
	}

	return fallback
}

func (state HandlerState) applicableClean(decodedPath string) bool {
	if len(state.CleanUrls) == 0 {
		return true
	}

	for _, source := range state.CleanUrls {
		if ok, _, _ := sourceMatches(source, decodedPath, false); ok {
			return true
		}
	}

	return false
}

func (state HandlerState) shouldRedirect(decodedPath string, cleanUrl bool) (*string, int) {
	slashing := false
	defaultType := http.StatusTemporaryRedirect

	if len(state.Redirects) == 0 && !slashing && !cleanUrl {
		return nil, defaultType
	}

	cleanedUrl := false

	// By stripping the HTML parts from the decoded
	// path *before* handling the trailing slash, we make
	// sure that only *one* redirect occurs if both
	// config options are used.
	if cleanUrl {
		if strings.HasSuffix(decodedPath, ".html") {
			decodedPath = decodedPath[:len(decodedPath)-5]
			cleanedUrl = true
		} else if strings.HasSuffix(decodedPath, "/index") {
			decodedPath = decodedPath[:len(decodedPath)-6]
			cleanedUrl = true
		}
	}

	if slashing {
		name := path.Base(decodedPath)
		ext := path.Ext(decodedPath)
		isTrailed := strings.HasSuffix(decodedPath, "/")
		isDotfile := strings.HasPrefix(name, ".")

		target := ""
		if state.TrailingSlash && isTrailed {
			target = decodedPath[0 : len(decodedPath)-1]
		} else if state.TrailingSlash && !isTrailed && ext == "" && !isDotfile {
			target = decodedPath + "/"
		}

		decodedPath = strings.ReplaceAll(decodedPath, "//", "/")

		if target != "" {
			return &target, defaultType
		}
	}

	if cleanedUrl {
		value := ensureSlashStart(decodedPath)
		return &value, defaultType
	}

	for _, item := range state.Redirects {
		target := toTarget(item.Source, item.Destination, decodedPath)

		if target != nil {
			if item.Type == 0 {
				return target, defaultType
			}
			return target, item.Type
		}
	}

	return nil, defaultType
}

func applicable(decodedPath string, configEntry []string, noFlag bool) bool {
	if noFlag {
		return false
	}
	if len(configEntry) == 0 {
		return true
	}

	for _, source := range configEntry {
		if ok, _, _ := sourceMatches(source, decodedPath, false); ok {
			return true
		}
	}

	return false
}

func (state HandlerState) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: Windows...
	relativePath := r.URL.Path
	absolutePath := filepath.Join(state.Public, relativePath)

	state.logger.Debug("Request =", relativePath)

	if !pathIsInside(absolutePath, state.Public) {
		state.sendError(w, r, "/", http.StatusBadRequest)
		return
	}

	cleanUrl := applicable(relativePath, state.CleanUrls, state.NoCleanUrls)
	redirect, _ := state.shouldRedirect(relativePath, cleanUrl)

	if redirect != nil {
		state.logger.Debug("Redirecting", redirect)
		http.Redirect(w, r, *redirect, http.StatusTemporaryRedirect)
		return
	}

	var stats os.FileInfo

	// It's extremely important that we're doing multiple stat calls. This one
	// right here could technically be removed, but then the program
	// would be slower. Because for directories, we always want to see if a related file
	// exists and then (after that), fetch the directory itself if no
	// related file was found. However (for files, of which most have extensions), we should
	// always stat right away.
	//
	// When simulating a file system without directory indexes, calculating whether a
	// directory exists requires loading all the file paths and then checking if
	// one of them includes the path of the directory. As that's a very
	// performance-expensive thing to do, we need to ensure it's not happening if not really necessary.

	if path.Ext(relativePath) != "" {
		fileInfo, err := os.Lstat(absolutePath)
		if err != nil && !os.IsNotExist(err) {
			state.sendError(w, r, "/", http.StatusBadRequest)
			return
		} else {
			stats = fileInfo
		}
	}

	rewrittenPath := applyRewrites(relativePath, state.Rewrites, false)

	if stats == nil && (cleanUrl || rewrittenPath != nil) {
		tstats, tabsolutePath := findRelated(state.Public, relativePath, rewrittenPath)
		if tstats != nil {
			stats = tstats
			absolutePath = tabsolutePath
		}
	}

	if stats == nil {
		fileInfo, err := os.Lstat(absolutePath)
		if err != nil && !os.IsNotExist(err) {
			state.sendError(w, r, "/", http.StatusBadRequest)
			return
		} else {
			stats = fileInfo
		}
	}

	if stats != nil && stats.IsDir() {
		related, err := state.renderDirectory(state.Public, relativePath, absolutePath)

		if err != nil {
			fmt.Println(err)
			state.sendError(w, r, "/", http.StatusInternalServerError)
			return
		}

		if related.singleFile {
			stats = related.stats
			absolutePath = related.absolutePath
		} else if related.outputData != nil {
			if acceptJSON(r) {
				if err := json.NewEncoder(w).Encode(related.outputData); err != nil {
					log.Fatal(err)
				}
			} else {
				if err := directoryTemplate.Execute(w, related.outputData); err != nil {
					log.Fatal(err)
				}
			}
			return
		} else {
			// The directory listing is disabled, so we want to
			// render a 404 error.
			stats = nil
		}
	}

	isSymLink := stats != nil && stats.Mode()&os.ModeSymlink == os.ModeSymlink

	// There are two scenarios in which we want to reply with
	// a 404 error: Either the path does not exist, or it is a
	// symlink while the `symlinks` option is disabled (which it is by default).
	if stats == nil || (!state.Symlinks && isSymLink) {
		state.sendError(w, r, "/", http.StatusNotFound)
		return
	}

	// If we figured out that the target is a symlink, we need to
	// resolve the symlink and run a new `stat` call just for the
	// target of that symlink.
	if isSymLink {
		var err error
		absolutePath, err = os.Readlink(absolutePath)
		if err != nil && !os.IsNotExist(err) {
			state.sendError(w, r, "/", http.StatusBadRequest)
			return
		}

		fileInfo, err := os.Lstat(absolutePath)
		if err != nil && !os.IsNotExist(err) {
			state.sendError(w, r, "/", http.StatusBadRequest)
			return
		} else {
			stats = fileInfo
		}
	}

	file, err := os.Open(absolutePath)
	if err != nil {
		state.sendError(w, r, "/", http.StatusBadRequest)
		return
	}

	http.ServeContent(w, r, absolutePath, stats.ModTime(), file)
}

func ensureSlashStart(target string) string {
	if len(target) > 0 && target[0] == '/' {
		return target
	}
	return "/" + target
}

func toTarget(source, destination, previousPath string) *string {
	didMatch, keys, results := sourceMatches(source, previousPath, true)

	if !didMatch {
		return nil
	}

	uinfo, err := url.Parse(destination)
	if err != nil {
		return nil
	}

	normalizedDest := destination
	if uinfo.Scheme != "" {
		normalizedDest = slasher(destination)
	}

	toPath := pathToRegExp.Compile(normalizedDest)

	props := map[string]string{}
	for index, item := range keys {
		props[item.Name] = results[index+1]
	}

	path := toPath(props)

	return &path
}

type fileDetails struct {
	Title    string
	Base     string
	Name     string
	Ext      string
	Dir      string
	Size     int
	Relative string
	IsDir    bool
}

type pathPart struct {
	Name string
	Url  string
}

type breadcrumbsType struct {
	Url  string
	Name string
}

type renderDirResult struct {
	singleFile   bool
	absolutePath string
	stats        os.FileInfo
	outputData   interface{}

	//directory    string
	//paths        []pathPart
	//files        []fileDetails
}

// const renderDirectory = async (current, acceptsJSON, handlers, methods, config, paths) => {
func (state HandlerState) renderDirectory(current string, relativePath string, absolutePath string) (renderDirResult, error) {
	trailingSlash := state.TrailingSlash
	unlisted := state.Unlisted
	renderSingle := state.RenderSingle

	slashSuffix := "/"
	if !trailingSlash {
		slashSuffix = ""
	}

	if !applicable(relativePath, state.DirectoryListing, state.NoDirectoryListing) {
		return renderDirResult{}, nil
	}

	files, err := ioutil.ReadDir(absolutePath)
	if err != nil {
		return renderDirResult{}, err
	}

	canRenderSingle := renderSingle && len(files) == 1

	fileResult := []fileDetails{}

	needSlash := "/"
	if len(relativePath) > 0 && relativePath[len(relativePath)-1] == '/' {
		needSlash = ""
	}

	for _, file := range files {
		if !canBeListed(unlisted, file.Name()) {
			continue
		}

		filePath := path.Join(absolutePath, file.Name())

		details := fileDetails{
			Base:     path.Base(file.Name()),
			Name:     file.Name(),
			Ext:      path.Ext(file.Name()),
			Dir:      path.Dir(file.Name()),
			IsDir:    file.IsDir(),
			Relative: relativePath + needSlash + file.Name(),
		}

		if file.IsDir() {
			details.Base += slashSuffix
			details.Relative += slashSuffix
		} else if canRenderSingle {
			return renderDirResult{
				singleFile:   true,
				absolutePath: filePath,
				stats:        file,
			}, nil
		}

		if details.Ext != "" {
			details.Ext = details.Ext[1:]
		} else {
			details.Ext = "txt"
		}

		// 			details.size = bytes(stats.size, {
		// 				unitSeparator: ' ',
		// 				decimalPlaces: 0
		// 			});
		// 		}
		details.Title = details.Base

		fileResult = append(fileResult, details)
	}

	// 	const toRoot = path.relative(current, absolutePath);
	// 	const directory = path.join(path.basename(current), toRoot, slashSuffix);
	// 	const pathParts = directory.split(path.sep).filter(Boolean);

	// 	// Sort to list directories first, then sort alphabetically
	// 	files = files.sort((a, b) => {
	// 		const aIsDir = a.type === 'directory';
	// 		const bIsDir = b.type === 'directory';

	// 		/* istanbul ignore next */
	// 		if (aIsDir && !bIsDir) {
	// 			return -1;
	// 		}

	// 		if ((bIsDir && !aIsDir) || (a.base > b.base)) {
	// 			return 1;
	// 		}

	// 		if (a.base < b.base) {
	// 			return -1;
	// 		}

	// 		/* istanbul ignore next */
	// 		return 0;
	// 	}).filter(Boolean);

	// 	// Add parent directory to the head of the sorted files array
	// 	if (toRoot.length > 0) {
	// 		const directoryPath = [...pathParts].slice(1);
	// 		const relative = path.join('/', ...directoryPath, '..', slashSuffix);

	// 		files.unshift({
	// 			type: 'directory',
	// 			base: '..',
	// 			relative,
	// 			title: relative,
	// 			ext: ''
	// 		});
	// 	}

	toRoot, err := filepath.Rel(current, absolutePath)
	if err != nil {
		return renderDirResult{}, err
	}
	directory := path.Join(filepath.Base(current), toRoot, slashSuffix)
	pathParts := strings.Split(relativePath, "/")

	fmt.Println(pathParts)

	breadcrumbs := []breadcrumbsType{
		{
			Name: strings.Split(directory, "/")[0],
			Url:  "/",
		},
	}
	parents := "/"

	for _, path := range pathParts[1 : len(pathParts)-1] {
		breadcrumbs = append(breadcrumbs, breadcrumbsType{
			Name: path,
			Url:  parents + path + "/",
		})

		parents += path + "/"
	}
	fmt.Println(breadcrumbs)

	type returnType struct {
		Directory string
		Index     []breadcrumbsType
		Paths     []pathPart
		Files     []fileDetails
	}

	return renderDirResult{
		outputData: returnType{
			Index:     breadcrumbs,
			Files:     fileResult,
			Directory: directory,
			// Paths:     subPaths,
		},
	}, nil
}

func canBeListed(excluded []string, file string) bool {
	slashed := slasher(file)

	for _, source := range excluded {
		if ok, _, _ := sourceMatches(source, slashed, false); ok {
			return false
		}
	}

	return true
}

func findRelated(current string, relativePath string, rewrittenPath *string) (os.FileInfo, string) {
	var possible []string

	if rewrittenPath == nil || *rewrittenPath == "" {
		possible = getPossiblePaths(relativePath, ".html")
	} else {
		possible = []string{*rewrittenPath}
	}

	for _, related := range possible {
		absolutePath := path.Join(current, related)

		stats, err := os.Lstat(absolutePath)

		if !os.IsNotExist(err) {
			return stats, absolutePath
		}
	}

	return nil, ""
}

func getPossiblePaths(relativePath, extension string) []string {
	entries := []string{
		path.Join(relativePath, "index"+extension),
	}
	part := relativePath
	if strings.HasSuffix(relativePath, "/") {
		part = relativePath[:len(relativePath)-1]
	}

	part = part + extension
	if path.Base(part) != extension {
		entries = append(entries, part)
	}

	return entries
}

func (state HandlerState) AttachRoutes(router chi.Router) {
	filesDir := http.Dir(state.Public)

	hasCatchall := false
	for _, item := range state.Proxy {
		router.Handle(item.Source, NewProxy(item.Destination))
		hasCatchall = hasCatchall || (item.Source == "/*")
	}
	// Default
	if !hasCatchall {
		router.Get("/*", state.sendFile(filesDir))
	}
}
