package server

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/dispatch"
	"github.com/named-data/YaNFD/face"
	"github.com/named-data/YaNFD/fw"
	"github.com/named-data/YaNFD/table"
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
	StatueMsg   string
	FibList     []routeBrief
	RibList     []routeBrief
	RequestName string
	FibHops     []routeHops
	RibHops     []routeHops
	Strategies  []strategy
}

var serverTmpl map[string]*template.Template

var menus = []menuList{
	{LinkName: "/forwarder-status", PageName: "Forwarder Status"},
	{LinkName: "/faces", PageName: "Faces"},
	{LinkName: "/routing", PageName: "Routing"},
	{LinkName: "/strategies", PageName: "Strategies"},
	{LinkName: "/autoconf", PageName: "Autoconfiguration"},
	{LinkName: "/key-management", PageName: "Key Management"},
	{LinkName: "/ndn-peek", PageName: "NDN Peek"},
}

var HttpBaseDir string = "cmd/yanfdui"

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
	http.ServeFile(w, req, HttpBaseDir+req.URL.Path)
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
	switch action {
	case "add":
		// TODO: add face
	case "remove":
		// TODO: remove face
	}

	input := renderInput{
		ReferName: "/faces",
		MenuList:  menus,
		FaceList:  make([]faceListEntry, 0),
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
	req.ParseForm()
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
	switch action {
	case "add":
		// TODO: add route
	case "remove":
		// TODO: remove route
	}

	input := renderInput{
		ReferName: "/routing",
		MenuList:  menus,
		FibList:   make([]routeBrief, 0),
		RibList:   make([]routeBrief, 0),
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
	req.ParseForm()
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
	switch action {
	case "set":
		// TODO: set strategies
	case "unset":
		// TODO: unset strategies
	}

	input := renderInput{
		ReferName:  "/strategies",
		MenuList:   menus,
		Strategies: make([]strategy, 0),
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

func StartHttpServer(wg *sync.WaitGroup, addr string) *http.Server {
	ret := &http.Server{Addr: addr}

	dir := HttpBaseDir + "/templates/"
	serverTmpl = make(map[string]*template.Template)
	serverTmpl["forwarder-status"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"forwarder-status.go.tmpl"))
	serverTmpl["index"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"index.go.tmpl"))
	serverTmpl["faces"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"faces.go.tmpl"))
	serverTmpl["routing"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"routing.go.tmpl"))
	serverTmpl["strategies"] = template.Must(template.ParseFiles(dir+"base.go.tmpl", dir+"strategies.go.tmpl"))

	http.HandleFunc("/", index)
	http.HandleFunc("/forwarder-status/", forwarderStatus)
	http.HandleFunc("/static/", statics)
	http.HandleFunc("/faces/", faces)
	http.HandleFunc("/routing/", routing)
	http.HandleFunc("/strategies/", strategies)

	go func() {
		defer wg.Done()

		if err := ret.ListenAndServe(); err != http.ErrServerClosed {
			core.LogError("HttpServer", "ListenAndServe() failed with: ", err)
		}
	}()

	return ret
}
