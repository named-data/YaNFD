package server

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/face"
	"github.com/named-data/YaNFD/fw"
	"github.com/named-data/YaNFD/table"
	"github.com/pelletier/go-toml"
)

type menuList struct {
	LinkName string
	PageName string
}

type faceListEntry struct {
	FaceID   uint64
	Uri      string
	LocalUri string
}

type faceData struct {
	FaceID        uint64
	Uri           string
	LocalUri      string
	Mtu           uint64
	NInInterests  uint64
	NInData       uint64
	NOutInterests uint64
	NOutData      uint64
}

type faceRouteData struct {
	Route string
	Cost  uint64
}

type routeBrief struct {
	Prefix     string
	RouteCount int
}

type routeHops struct {
	FaceID uint64
	Uri    string
	Cost   uint64
	Origin uint64
	Flags  uint64
}

type strategy struct {
	Name     string
	Strategy string
}

type renderInput struct {
	ReferName   string
	MenuList    []menuList
	Status      map[string]string
	FaceList    []faceListEntry
	FaceData    *faceData
	RouteData   []faceRouteData
	StatusCode  int
	StatusMsg   string
	FibList     []routeBrief
	RibList     []routeBrief
	RequestName string
	FibHops     []routeHops
	RibHops     []routeHops
	Strategies  []strategy
	Setting     setting
}

// TODO: setting is a temporary solution; should change to toml.Marshal
type setting struct {
	LogLevel                 string `toml:"core.log_level"`
	FacesQueueSize           int    `toml:"faces.queue_size"`
	FacesCongestionMarking   bool   `toml:"faces.congestion_marking"`
	FacesLockThreadsToCores  bool   `toml:"faces.lock_threads_to_cores"`
	EtherEnabled             bool   `toml:"faces.ethernet.enabled"`
	EtherType                int    `toml:"faces.ethernet.ethertype"`
	EtherAddr                string `toml:"faces.ethernet.multicast_address"`
	UdpPortUnicast           uint16 `toml:"faces.udp.port_unicast"`
	UdpPortMulticast         uint16 `toml:"faces.udp.port_multicast"`
	UdpMulticastIpv4         string `toml:"faces.udp.multicast_address_ipv4"`
	UdpMulticastIpv6         string `toml:"faces.udp.multicast_address_ipv6"`
	UdpLifetime              uint16 `toml:"faces.udp.lifetime"`
	TcpEnabled               bool   `toml:"faces.tcp.enabled"`
	TcpPort                  uint16 `toml:"faces.tcp.port_unicast"`
	TcpLifetime              uint16 `toml:"faces.tcp.lifetime"`
	UnixEnabled              bool   `toml:"faces.unix.enabled"`
	UnixSocketPath           string `toml:"faces.unix.socket_path"`
	WsEnabled                bool   `toml:"faces.websocket.enabled"`
	WsBind                   string `toml:"faces.websocket.bind"`
	WsPort                   uint16 `toml:"faces.websocket.port"`
	WsTlsEnabled             bool   `toml:"faces.websocket.tls_enabled"`
	WsTlsCert                string `toml:"faces.websocket.tls_cert"`
	WsTlsKey                 string `toml:"faces.websocket.tls_key"`
	FwThreads                int    `toml:"fw.threads"`
	FwQueueSize              int    `toml:"fw.queue_size"`
	FwLockThreadsToCores     bool   `toml:"fw.lock_threads_to_cores"`
	AllowLocalhop            bool   `toml:"mgmt.allow_localhop"`
	TablesQueueSize          int    `toml:"tables.queue_size"`
	CsCapacity               uint16 `toml:"tables.content_store.capacity"`
	CsAdmit                  bool   `toml:"tables.content_store.admit"`
	CsServe                  bool   `toml:"tables.content_store.serve"`
	CsReplacementPolicy      string `toml:"tables.content_store.replacement_policy"`
	DnlLifetime              int    `toml:"tables.dead_nonce_list.lifetime"`
	RibAutoPrefixPropagation bool   `toml:"tables.rib.auto_prefix_propagation"`
}

var (
	serverTmpl  map[string]*template.Template
	httpBaseDir string
	configFile  string
)

var menus = []menuList{
	{LinkName: "/forwarder-status", PageName: "Forwarder Status"},
	{LinkName: "/faces", PageName: "Faces"},
	{LinkName: "/routing", PageName: "Routing"},
	{LinkName: "/strategies", PageName: "Strategies"},
	{LinkName: "/config", PageName: "Configuration"},
}

func tomlTreeToSetting(tree *toml.Tree) *setting {
	var s setting
	sType := reflect.TypeOf(s)
	for i := 0; i < sType.NumField(); i++ {
		f := sType.Field(i)
		if fName := f.Tag.Get("toml"); fName != "" {
			switch f.Type.Kind() {
			case reflect.String:
				if v, ok := tree.Get(fName).(string); ok {
					reflect.ValueOf(&s).Elem().Field(i).SetString(v)
				}
			case reflect.Uint16:
				if v, ok := tree.Get(fName).(int64); ok {
					reflect.ValueOf(&s).Elem().Field(i).SetUint(uint64(v))
				}
			case reflect.Int:
				if v, ok := tree.Get(fName).(int64); ok {
					reflect.ValueOf(&s).Elem().Field(i).SetInt(v)
				}
			case reflect.Bool:
				if v, ok := tree.Get(fName).(bool); ok {
					reflect.ValueOf(&s).Elem().Field(i).SetBool(v)
				}
			}
		}
	}
	return &s
}

func tomlFormToTree(tree *toml.Tree, r *http.Request) {
	sType := reflect.TypeOf(setting{})
	for i := 0; i < sType.NumField(); i++ {
		f := sType.Field(i)
		val := r.Form.Get(f.Name)
		if val == "" {
			continue
		}
		if fName := f.Tag.Get("toml"); fName != "" {
			switch f.Type.Kind() {
			case reflect.String:
				tree.Set(fName, val)
			case reflect.Uint16:
				v, err := strconv.ParseUint(val, 10, 16)
				if err != nil {
					tree.Set(fName, v)
				}
			case reflect.Int:
				v, err := strconv.Atoi(val)
				if err != nil {
					tree.Set(fName, v)
				}
			case reflect.Bool:
				if strings.ToUpper(val) == "TRUE" {
					tree.Set(fName, true)
				} else {
					tree.Set(fName, false)
				}
			}
		}
	}
}

func forwarderStatus(w http.ResponseWriter, req *http.Request) {
	var nPitEntries, nCsEntries, nInInterests, nInData, nOutInterests, nFibEntries uint64
	var nOutData, nSatisfiedInterests, nUnsatisfiedInterests uint64

	nFibEntries = uint64(len(table.FibStrategyTable.GetAllFIBEntries()))
	for threadID := 0; threadID < fw.NumFwThreads; threadID++ {
		thread := dispatch.GetFWThread(threadID)
		nPitEntries += uint64(thread.GetNumPitEntries())
		nCsEntries += uint64(thread.GetNumCsEntries())
		nInInterests += thread.(*fw.Thread).NInInterests
		nInData += thread.(*fw.Thread).NInData
		nOutInterests += thread.(*fw.Thread).NOutInterests
		nOutData += thread.(*fw.Thread).NOutData
		nSatisfiedInterests += thread.(*fw.Thread).NSatisfiedInterests
		nUnsatisfiedInterests += thread.(*fw.Thread).NUnsatisfiedInterests
	}

	input := renderInput{
		ReferName: "/forwarder-status",
		MenuList:  menus,
		Status: map[string]string{
			"YaNFD Version":      core.Version,
			"Start Time":         core.StartTimestamp.Local().String(),
			"Current Time":       time.Now().Local().String(),
			"Entries FIB PIT CS": fmt.Sprint("fib=", nFibEntries, " pit=", nPitEntries, " cs=", nCsEntries),
			"Counter IN":         fmt.Sprint(nInInterests, "i ", nInData, "d"),
			"Counter OUT":        fmt.Sprint(nOutInterests, "i ", nOutData, "d"),
			"INT Sat/Unsat":      fmt.Sprint(nSatisfiedInterests, " / ", nUnsatisfiedInterests),
		},
	}
	if err := serverTmpl["forwarder-status"].ExecuteTemplate(w, "forwarder-status.go.tmpl", input); err != nil {
		core.LogError("HttpServer", "forwarderStatus.Execute failed with: ", err)
	}
}

func statics(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, httpBaseDir+req.URL.Path)
}

func notImplemented(w http.ResponseWriter, req *http.Request, url string) {
	http.Redirect(w, req, url+"?status_code=-1&status_text=Not%20Implemented", http.StatusFound)
}

func index(w http.ResponseWriter, req *http.Request) {
	input := renderInput{
		ReferName: "/",
		MenuList:  menus,
	}
	if err := serverTmpl["index"].ExecuteTemplate(w, "index.go.tmpl", input); err != nil {
		core.LogError("HttpServer", "index.Execute failed with: ", err)
	}
}

func faces(w http.ResponseWriter, req *http.Request) {
	action := req.URL.Path[len("/faces/"):]
	req.ParseForm()
	switch action {
	case "add":
		// TODO: add face
		notImplemented(w, req, "/faces/")
	case "remove":
		// TODO: remove face
		notImplemented(w, req, "/faces/")
	}

	input := renderInput{
		ReferName: "/faces",
		MenuList:  menus,
		FaceList:  make([]faceListEntry, 0),
	}
	if stCodeStr := req.Form.Get("status_code"); stCodeStr != "" {
		stCode, err := strconv.Atoi(stCodeStr)
		if err == nil {
			input.StatusCode = stCode
			input.StatusMsg = req.Form.Get("status_text")
		}
	}

	// Obtain face lists
	faces := face.FaceTable.GetAll()
	// Sort face list by ID
	sort.Slice(faces, func(i, j int) bool {
		return faces[i].FaceID() < faces[j].FaceID()
	})
	for _, face := range faces {
		input.FaceList = append(input.FaceList, faceListEntry{
			FaceID:   face.FaceID(),
			Uri:      face.RemoteURI().String(),
			LocalUri: face.LocalURI().String(),
		})
	}

	// Obtain face data and associated routes
	func(faceIdStr string) {
		if faceIdStr == "" {
			return
		}
		fid, err := strconv.ParseUint(faceIdStr, 10, 64)
		if err != nil {
			return
		}
		for _, face := range faces {
			if face.FaceID() == fid {
				input.FaceData = &faceData{
					FaceID:        face.FaceID(),
					Uri:           face.RemoteURI().String(),
					LocalUri:      face.LocalURI().String(),
					Mtu:           uint64(face.MTU()),
					NInInterests:  face.NInInterests(),
					NInData:       face.NInData(),
					NOutInterests: face.NOutInterests(),
					NOutData:      face.NOutData(),
				}
				break
			}
		}
		if input.FaceData == nil {
			return
		}
		input.RouteData = make([]faceRouteData, 0)
		for _, entry := range table.FibStrategyTable.GetAllFIBEntries() {
			for _, nh := range entry.GetNexthops() {
				if nh.Nexthop == fid {
					input.RouteData = append(input.RouteData, faceRouteData{
						Route: entry.Name.String(),
						Cost:  nh.Cost,
					})
				}
			}
		}
		sort.Slice(input.RouteData, func(i, j int) bool {
			return input.RouteData[i].Route < input.RouteData[j].Route
		})
	}(req.Form.Get("face_id"))

	// Render
	if err := serverTmpl["faces"].ExecuteTemplate(w, "faces.go.tmpl", input); err != nil {
		core.LogError("HttpServer", "faces.Execute failed with: ", err)
	}
}

func routing(w http.ResponseWriter, req *http.Request) {
	action := req.URL.Path[len("/routing/"):]
	req.ParseForm()
	switch action {
	case "add":
		// TODO: add route
		notImplemented(w, req, "/routing/")
	case "remove":
		// TODO: remove route
		notImplemented(w, req, "/routing/")
	}

	input := renderInput{
		ReferName: "/routing",
		MenuList:  menus,
		FibList:   make([]routeBrief, 0),
		RibList:   make([]routeBrief, 0),
	}
	if stCodeStr := req.Form.Get("status_code"); stCodeStr != "" {
		stCode, err := strconv.Atoi(stCodeStr)
		if err == nil {
			input.StatusCode = stCode
			input.StatusMsg = req.Form.Get("status_text")
		}
	}

	// Obtain route lists
	fib := table.FibStrategyTable.GetAllFIBEntries()
	rib := table.Rib.GetAllEntries()
	for _, entry := range fib {
		input.FibList = append(input.FibList, routeBrief{
			Prefix:     entry.Name.String(),
			RouteCount: len(entry.GetNexthops()),
		})
	}
	for _, entry := range rib {
		input.RibList = append(input.RibList, routeBrief{
			Prefix:     entry.Name.String(),
			RouteCount: len(entry.GetRoutes()),
		})
	}
	// Sort route lists
	sort.Slice(input.FibList, func(i, j int) bool {
		return input.FibList[i].Prefix < input.FibList[j].Prefix
	})
	sort.Slice(input.RibList, func(i, j int) bool {
		return input.RibList[i].Prefix < input.RibList[j].Prefix
	})

	// Obtain associated routes
	func(prefix string) {
		if prefix == "" {
			return
		}
		input.RequestName = prefix
		faceMap := make(map[uint64]string)
		for _, face := range face.FaceTable.GetAll() {
			faceMap[face.FaceID()] = face.RemoteURI().String()
		}
		for _, entry := range fib {
			if entry.Name.String() == prefix {
				input.FibHops = make([]routeHops, 0)
				for _, nh := range entry.GetNexthops() {
					input.FibHops = append(input.FibHops, routeHops{
						FaceID: nh.Nexthop,
						Uri:    faceMap[nh.Nexthop],
						Cost:   nh.Cost,
					})
				}
				break
			}
		}
		for _, entry := range rib {
			if entry.Name.String() == prefix {
				input.RibHops = make([]routeHops, 0)
				for _, nh := range entry.GetRoutes() {
					input.RibHops = append(input.FibHops, routeHops{
						FaceID: nh.FaceID,
						Uri:    faceMap[nh.FaceID],
						Cost:   nh.Cost,
						Origin: nh.Origin,
						Flags:  nh.Flags,
					})
				}
				break
			}
		}
	}(req.Form.Get("name"))

	// Render
	if err := serverTmpl["routing"].ExecuteTemplate(w, "routing.go.tmpl", input); err != nil {
		core.LogError("HttpServer", "routing.Execute failed with: ", err)
	}
}

func strategies(w http.ResponseWriter, req *http.Request) {
	action := req.URL.Path[len("/strategies/"):]
	req.ParseForm()
	switch action {
	case "set":
		// TODO: set strategies
		notImplemented(w, req, "/strategies/")
	case "unset":
		// TODO: unset strategies
		notImplemented(w, req, "/strategies/")
	}

	input := renderInput{
		ReferName:  "/strategies",
		MenuList:   menus,
		Strategies: make([]strategy, 0),
	}
	if stCodeStr := req.Form.Get("status_code"); stCodeStr != "" {
		stCode, err := strconv.Atoi(stCodeStr)
		if err == nil {
			input.StatusCode = stCode
			input.StatusMsg = req.Form.Get("status_text")
		}
	}

	// Obtain strategy list
	for _, sc := range table.FibStrategyTable.GetAllStrategyChoices() {
		input.Strategies = append(input.Strategies, strategy{
			Name:     sc.Name.String(),
			Strategy: sc.GetStrategy().String(),
		})
	}

	// Render
	if err := serverTmpl["strategies"].ExecuteTemplate(w, "strategies.go.tmpl", input); err != nil {
		core.LogError("HttpServer", "strategies.Execute failed with: ", err)
	}
}

func autoconf(w http.ResponseWriter, req *http.Request) {
	action := req.URL.Path[len("/autoconf/"):]
	req.ParseForm()
	switch action {
	case "perform":
		// TODO: perform autoconfiguration
		notImplemented(w, req, "/autoconf/")
	}

	input := renderInput{
		ReferName: "/autoconf",
		MenuList:  menus,
	}
	if stCodeStr := req.Form.Get("status_code"); stCodeStr != "" {
		stCode, err := strconv.Atoi(stCodeStr)
		if err == nil {
			input.StatusCode = stCode
			input.StatusMsg = req.Form.Get("status_text")
		}
	}

	// Render
	if err := serverTmpl["autoconf"].ExecuteTemplate(w, "autoconf.go.tmpl", input); err != nil {
		core.LogError("HttpServer", "autoconf.Execute failed with: ", err)
	}
}

func config(w http.ResponseWriter, req *http.Request) {
	action := req.URL.Path[len("/config/"):]
	req.ParseForm()

	tree, err := toml.LoadFile(configFile)
	if err != nil {
		core.LogError("HttpServer", "toml.LoadFile failed with: ", err)
		// TODO: handle error
		http.Redirect(w, req, "/", http.StatusFound)
		return
	}

	switch action {
	case "save":
		tomlFormToTree(tree, req)
		s, err := tree.ToTomlString()
		if err == nil {
			file, err := os.Create(configFile)
			if err == nil {
				file.WriteString(s)
			} else {
				core.LogError("HttpServer", "os.Create failed with: ", err)
			}
		} else {
			core.LogError("HttpServer", "toml.ToTomlString failed with: ", err)
		}
		http.Redirect(w, req, "/config/", http.StatusFound)
		return
	}

	set := tomlTreeToSetting(tree)

	input := renderInput{
		ReferName: "/",
		MenuList:  menus,
		Setting:   *set,
	}

	// Render
	if err := serverTmpl["config"].ExecuteTemplate(w, "config.go.tmpl", input); err != nil {
		core.LogError("HttpServer", "config.Execute failed with: ", err)
	}
}

func StartHttpServer(wg *sync.WaitGroup, addr string, baseDir string, configFilePath string) *http.Server {
	ret := &http.Server{Addr: addr}

	httpBaseDir = baseDir
	configFile = configFilePath
	dir := httpBaseDir + "/templates/"
	serverTmpl = make(map[string]*template.Template)
	serverTmpl["forwarder-status"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"forwarder-status.go.tmpl"))
	serverTmpl["index"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"index.go.tmpl"))
	serverTmpl["faces"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"faces.go.tmpl"))
	serverTmpl["routing"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"routing.go.tmpl"))
	serverTmpl["strategies"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"strategies.go.tmpl"))
	serverTmpl["autoconf"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"autoconf.go.tmpl"))
	serverTmpl["config"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"config.go.tmpl"))

	http.HandleFunc("/", index)
	http.HandleFunc("/forwarder-status/", forwarderStatus)
	http.HandleFunc("/static/", statics)
	http.HandleFunc("/faces/", faces)
	http.HandleFunc("/routing/", routing)
	http.HandleFunc("/strategies/", strategies)
	http.HandleFunc("/autoconf/", autoconf)
	http.HandleFunc("/config/", config)

	go func() {
		defer wg.Done()

		if err := ret.ListenAndServe(); err != http.ErrServerClosed {
			core.LogError("HttpServer", "ListenAndServe() failed with: ", err)
		}
	}()

	return ret
}

func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		core.LogError("HttpServer", "Unable to open the browser on OS: ", runtime.GOOS)
		return nil
	}
}
