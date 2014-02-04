// custard is a standalone compositing manager for use with window managers
// without their own compositors.  The code here is based off Compton, but
// with the use of XCB (XGB by BurntSushi) instead of Xlib.  I have tried
// to use GLFW3 for the opengl work here.  I also wanted to
// learn the GO programing language at the same time.

package main

import (
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/composite"
	"github.com/BurntSushi/xgb/shape"
	"github.com/BurntSushi/xgb/xfixes"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	//"github.com/go-gl/glfw3"
	"github.com/remogatto/egl"
	"runtime"
)

//proof is in the pudding, holds all the ingredients for custard
type pudding struct {
	dpy        *xgb.Conn
	root       xproto.Window
	setup      *xproto.SetupInfo
	overlay    xproto.Window
	eglOverlay egl.NativeWindowType
	//overlay *glfw3.Window
	//monitor *glfw3.Monitor
	util    *xgbutil.XUtil
	dpyName string
	err     error
	created bool
	glfw    bool
}

// holds the flavors if we want to get fancy
type setup struct {
	shadow bool
}

func overlay_init(prime *pudding) {
	var replytemp *composite.GetOverlayWindowReply
	cookietemp := composite.GetOverlayWindow(prime.dpy, prime.root)
	replytemp, prime.err = cookietemp.Reply()
	if prime.err != nil {
		panic(prime.err)
	}
	prime.overlay = replytemp.OverlayWin
	// now I need to let inputs pass through the overlay
	var region xfixes.Region
	// make a nil region
	regionCookie := xfixes.CreateRegion(prime.dpy, region, nil)
	prime.err = regionCookie.Check()
	if prime.err != nil {
		panic(prime.err)
	}
	// set bounding region to default? and make input region to nil
	// shape kind: Bounding (0), Cliping (1), Input (2)
	xfixes.SetWindowShapeRegion(prime.dpy, prime.overlay, 0, 0, 0, 0)
	xfixes.SetWindowShapeRegion(prime.dpy, prime.overlay, 2, 0, 0, region)
	xfixes.DestroyRegion(prime.dpy, region)
	shape.SelectInput(prime.dpy, prime.overlay, true)
}

func findWindow(prime *pudding, win xproto.Window) bool {
	return false
}

func addWindow(prime *pudding, child []xproto.Window, i uint16) {
	if child[i] == prime.overlay || findWindow(prime, child[i]) {
		return
	}

}

// kitchen prep and gather ingredients
func pudding_init(prime *pudding) pudding {
	// create a new connection passing a Null display name
	prime.dpy, prime.err = xgb.NewConnDisplay(prime.dpyName)
	if prime.err != nil {
		panic(prime.err)
	}
	// need SetupInfo (setuptemp) type to send to ScreenInfo to get the root
	prime.setup = xproto.Setup(prime.dpy)
	prime.root = prime.setup.DefaultScreen(prime.dpy).Root
	// make overlay window
	go overlay_init(prime)
	// lock the X Window System so nothing changes as we setup
	prime.util.Grab()
	var treeReply *xproto.QueryTreeReply
	treeCookie := xproto.QueryTree(prime.dpy, prime.root)
	treeReply, prime.err = treeCookie.Reply()
	if prime.err != nil {
		panic(prime.err)
	}
	var i uint16
	for i = 0; i < treeReply.ChildrenLen; i++ {
		addWindow(prime, treeReply.Children, i)
	}
	// let go of the X Window System so others can play
	prime.util.Ungrab()
	prime.created = true
	return *prime
}

// Let's make some custard
func pudding_stir(prime *pudding) {

}

// eat the pudding
func pudding_eat(prime *pudding) {
	//glfw3.Terminate()
}

// meat and potatoes, no thanks.  Let's make some custard
func main() {
	// We have more than 1 CPU, so let's use them
	runtime.GOMAXPROCS(runtime.NumCPU())
	i := true
	var prev, current pudding
	//go glfw3.Init()
	go xfixes.Init(prev.dpy)
	go composite.Init(prev.dpy)
	go shape.Init(prev.dpy)
	for i {
		// get ingredients for pudding
		current = pudding_init(&prev)
		// panic if we're out of eggs
		if !current.created {
			panic("error on custard initialization")
		}
		pudding_stir(&current)
		// if we get tired of stirring, go back to initial setup and go again
		prev = current
		// throw out the bad ingredients
		pudding_eat(&current)
	}
}
