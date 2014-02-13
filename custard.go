/*
The MIT License (MIT)

Copyright (c) 2014 Eric Trantina

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

/* custard is a standalone compositing manager for use with window managers
without their own compositors(e.g. OpenBox).  The code here is based off
Compton, but with the use of XCB (XGB by BurntSushi) instead of Xlib.
I have tried to use OpenGLES with EGL to completely forgo Xlib and GLX.
This was done partly to use all native Go code as well as to avoid having to
do the ugly mix of Xlib and XCB to use OpenGL.  I also wanted to learn
the GO programing language at the same time.
*/

package main

import (
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/composite"
	"github.com/BurntSushi/xgb/shape"
	"github.com/BurntSushi/xgb/xfixes"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/chsc/gogl/gl43"
	//"github.com/chsc/gogl2/gles2/2.0/gles2"
	"github.com/remogatto/egl"
	"runtime"
)

//proof is in the pudding, holds all the ingredients for custard
type pudding struct {
	dpy                *xgb.Conn
	egldpy             egl.Display
	eglconf            egl.Config
	eglconf_size       int32
	eglnum_conf        int32
	eglattrlist        []int32
	eglattrlist2       int32
	eglSurf            egl.Surface
	eglContext         egl.Context
	EGL_CLIENT_VERSION int32
	framebuffer        gl43.Uint
	colorbuffer        gl43.Uint
	root               xproto.Window
	setup              *xproto.SetupInfo
	width              uint16
	eglwidth           int32
	glwidth            gl43.Sizei
	glheight           gl43.Sizei
	eglheight          int32
	height             uint16
	overlay            xproto.Window
	eglOverlay         egl.NativeWindowType
	util               *xgbutil.XUtil
	list               []xproto.Window
	dpyName            string
	err                error
	created            bool
	glfw               bool
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
	xfixes.SetWindowShapeRegion(prime.dpy, prime.overlay, shape.SkBounding, 0, 0, 0)
	xfixes.SetWindowShapeRegion(prime.dpy, prime.overlay, shape.SkInput, 0, 0, region)
	xfixes.DestroyRegion(prime.dpy, region)
	shape.SelectInput(prime.dpy, prime.overlay, true)
	prime.eglOverlay = egl.NativeWindowType(uintptr(prime.overlay))
}

func findWindow(prime *pudding, win xproto.Window) bool {
	for i := 0; i < len(prime.list); i++ {
		if win == prime.list[i] {
			return true
		}
	}
	return false
}

func addWindow(prime *pudding, child []xproto.Window, i uint16) {
	attrcookie := xproto.GetWindowAttributes(prime.dpy, child[i])
	var attrReply *xproto.GetWindowAttributesReply
	if child[i] == prime.overlay || findWindow(prime, child[i]) {
		return
	}
	attrReply, prime.err = attrcookie.Reply()
	if prime.err != nil || attrReply.MapState == xproto.MapStateUnviewable {
		return
	}
	mapState := attrReply.MapState
	if mapState == xproto.MapStateUnmapped || mapState == xproto.MapStateUnviewable {
		return
	}
	attrReply.MapState = xproto.MapStateUnmapped
	if attrReply.Class == xproto.WindowClassInputOutput {
		//To DO
	}
}

func loadShader() bool {
	//Vertex Shader Program
	vertSourceString := "testing;" +
		"void main()" + "{" +
		"}"

	//Fragment Shader Program
	fragSourceString := "testing;" + "testing;" +
		"void main()" + "{" +
		"}"

	vertSource := gl43.GLString(vertSourceString)
	fragSource := gl43.GLString(fragSourceString)

	var vertShader, fragShader, program gl43.Uint
	test1 := gl43.IsShader(vertShader)
	test2 := gl43.IsShader(fragShader)
	test3 := gl43.IsProgram(program)
	// create shader program
	program = gl43.CreateProgram()
	vertShader = gl43.CreateShader(gl43.VERTEX_SHADER)
	fragShader = gl43.CreateShader(gl43.FRAGMENT_SHADER)
	gl43.ShaderSource(vertShader, 1, &vertSource, nil)
	gl43.ShaderSource(fragShader, 1, &fragSource, nil)
	gl43.CompileShader(vertShader)
	gl43.CompileShader(fragShader)

	if gl43.IsShader(vertShader) == test1 {
		panic("vert shader did not compile")
	}
	if gl43.IsShader(fragShader) == test2 {
		panic("frag shader did not compile")
	}
	gl43.AttachShader(program, vertShader)
	gl43.AttachShader(program, fragShader)
	gl43.LinkProgram(program)
	if gl43.IsProgram(program) == test3 {
		gl43.DeleteShader(vertShader)
		gl43.DeleteShader(fragShader)
		gl43.DeleteProgram(program)
		panic("program did not link")
	}

	gl43.DeleteShader(vertShader)
	gl43.DeleteShader(fragShader)
	gl43.ReleaseShaderCompiler()

	return true
}

func openglInit(prime *pudding) {
	// create the default framebuffer
	gl43.GenFramebuffers(1, &prime.framebuffer)
	gl43.BindFramebuffer(gl43.FRAMEBUFFER, prime.framebuffer)
	// create the default renderbuffer
	gl43.GenRenderbuffers(1, &prime.colorbuffer)
	gl43.BindRenderbuffer(gl43.RENDERBUFFER, prime.colorbuffer)
	//TO DO need to define glwidth and glheight
	gl43.RenderbufferStorage(gl43.RENDERBUFFER, gl43.RGBA, prime.glwidth, prime.glheight)
	// associate the framebuffer with the Renderbuffer
	gl43.FramebufferRenderbuffer(gl43.FRAMEBUFFER, gl43.COLOR_ATTACHMENT0, gl43.RENDERBUFFER, prime.colorbuffer)
	if gl43.CheckFramebufferStatus(gl43.FRAMEBUFFER) != gl43.FRAMEBUFFER_COMPLETE {
		panic("Framebuffer did not associate with the Renderbuffer")
	}
	if !loadShader() {
		panic("Shader did not load properly")
	}

}

// kitchen prep and gather ingredients
func pudding_init(prime *pudding) pudding {
	// create a new connection passing a Null display name
	prime.dpy, prime.err = xgb.NewConnDisplay(prime.dpyName)
	if prime.err != nil {
		panic(prime.err)
	}
	// Create the EGL display
	prime.egldpy = egl.GetDisplay(egl.DEFAULT_DISPLAY)
	if prime.egldpy == egl.NO_DISPLAY {
		panic("no egl display created")
	}
	//initialize EGL
	if !egl.Initialize(prime.egldpy, nil, nil) {
		panic("EGL did not initialize")
	}
	//get the configs for the display

	// TO DO: Determine correct configs.  Time to read up

	if !egl.GetConfigs(prime.egldpy, &prime.eglconf, prime.eglconf_size, &prime.eglnum_conf) {
		panic("Could not get egl configs")
	}
	//set configs with choose configs
	if !egl.ChooseConfig(prime.egldpy, prime.eglattrlist, &prime.eglconf, prime.eglconf_size, &prime.eglnum_conf) {
		panic("Could not get egl configs")
	}
	// need SetupInfo (setuptemp) type to send to ScreenInfo to get the root
	prime.setup = xproto.Setup(prime.dpy)
	prime.root = prime.setup.DefaultScreen(prime.dpy).Root
	prime.width = prime.setup.DefaultScreen(prime.dpy).WidthInPixels
	prime.height = prime.setup.DefaultScreen(prime.dpy).HeightInPixels
	// make overlay window
	overlay_init(prime)
	// create the window surface to draw to
	prime.eglSurf = egl.CreateWindowSurface(prime.egldpy, prime.eglconf, prime.eglOverlay, &prime.eglattrlist2)
	if prime.eglSurf == egl.NO_SURFACE {
		panic("EGL Surface failed to create")
	}
	egl.QuerySurface(prime.egldpy, prime.eglSurf, egl.WIDTH, &prime.eglwidth)
	egl.QuerySurface(prime.egldpy, prime.eglSurf, egl.HEIGHT, &prime.eglheight)
	// create EGL context and then make current with the surface
	prime.eglContext = egl.CreateContext(prime.egldpy, prime.eglconf, egl.NO_CONTEXT, &prime.EGL_CLIENT_VERSION)
	if prime.eglContext == egl.NO_CONTEXT {
		panic("EGL Context failed to create")
	}
	//if prime.EGL_CLIENT_VERSION == 1 {
	//	panic("Opengl ES version 1.X context was created, not version 2.0")
	//}
	if !egl.MakeCurrent(prime.egldpy, prime.eglSurf, egl.NO_SURFACE, prime.eglContext) {
		panic("EGL was not able to make context current with surface")
	}
	// Initialize the opengl portion
	go openglInit(prime)
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
	go gl43.Init()
	//go gles2.Init()
	go xfixes.Init(prev.dpy)
	go composite.Init(prev.dpy)
	go shape.Init(prev.dpy)
	go egl.BindAPI(egl.OPENGL_API)
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
