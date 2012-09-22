package Engine

import (
	"github.com/vova616/gl"
	//"log"
	. "github.com/vova616/GarageEngine/Engine/Input"
	//"os"
	"fmt"
	"github.com/jteeuwen/glfw"
	c "github.com/vova616/chipmunk"
	. "github.com/vova616/chipmunk/vect"
	"math"
	"runtime"
	"time"
)

func init() {
	fmt.Print()
}

const (
	RadianConst = math.Pi / 180
	DegreeConst = 180 / math.Pi
)

var (
	Inf = Float(math.Inf(1))

	scenes       []Scene = make([]Scene, 0)
	activeScenes []Scene = make([]Scene, 0)
	mainScene    Scene
	running               = false
	Space        *c.Space = c.NewSpace()
	deltaTime    float32
	fixedTime    float32
	stepTime     = float32(1) / float32(60)

	EnablePhysics = true
	Debug         = false
	InternalFPS   = float32(100)

	Title  = "Engine Test"
	Width  = 1024
	Height = 768

	terminated chan bool
)

func init() {
	terminated = make(chan bool)

}

func LoadScene(scene Scene) {
	sn := scene.New()
	sn.Load()

	internalFPS := NewGameObject("InternalFPS")
	internalFPS.AddComponent(NewFPS())
	sn.SceneBase().AddGameObject(internalFPS)

	mainScene = sn
}

func GetScene() Scene {
	return mainScene
}

func AddScene(scene Scene) {
	scenes = append(scenes, scene)
}

func ShutdownRecived() {
	terminated <- true
}

func Terminated() {
	<-terminated
}

func Terminate() {
	glfw.Terminate()
	ShutdownRecived()
}

func DeltaTime() float32 {
	return deltaTime
}

func StartEngine() {
	runtime.LockOSThread()
	println("Enginge started!")
	var err error
	if err = glfw.Init(); err != nil {
		panic(err)
	}

	if err = glfw.OpenWindow(Width, Height, 8, 8, 8, 8, 8, 8, glfw.Windowed); err != nil {
		panic(err)
	}

	glfw.SetSwapInterval(1) //0 to make FPS Maximum
	glfw.SetWindowTitle(Title)
	glfw.SetWindowSizeCallback(onResize)
	glfw.SetKeyCallback(OnKey)
	glfw.SetMouseButtonCallback(ButtonPress)

	if err = initGL(); err != nil {
		panic(err)
	}

	TextureMaterial = NewBasicMaterial(vertexShader, fragmentShader)
	TextureMaterial.Load()
}

func MainLoop() bool {
	running = true

	if running && glfw.WindowParam(glfw.Opened) == 1 {
		Run()
	} else {
		return false
	}
	return true
}

func Run() {
	before := time.Now()

	gl.ClearColor(0, 0, 0, 0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.LoadIdentity()

	if mainScene != nil {
		fixedTime += deltaTime
		sd := mainScene.SceneBase()

		arr := sd.gameObjects

		timer := NewTimer()
		timer.Start()

		timer.StartCustom("Destory routines")
		Iter(arr, destoyGameObject)
		destroyDelta := timer.StopCustom("Destory routines")

		timer.StartCustom("Start routines")
		Iter(arr, startGameObject)
		startDelta := timer.StopCustom("Start routines")

		//for fixedTime > stepTime {
		timer.StartCustom("FixedUpdate routines")
		Iter(arr, fixedUdpateGameObject)
		fixedUpdateDelta := timer.StopCustom("FixedUpdate routines")

		timer.StartCustom("Physics time")
		if EnablePhysics {
			for _, b := range Space.AllBodies {
				g, ok := b.UserData.(*Physics)
				if ok && g != nil {
					pos := g.Transform().WorldPosition()
					b.SetAngle(Float(180-g.Transform().WorldRotation().Z) * RadianConst)
					//fmt.Println(g.Transform().WorldRotation().Z, b.Transform.Angle())
					b.SetPosition(Vect{Float(pos.X), Float(pos.Y)})
					*g.lastCollision = *g.currentCollision
					g.currentCollision.ShapeA = nil
					g.currentCollision.ShapeB = nil
				}
			}

			Space.Step(Float(stepTime))
			//Space.Step(Float(0.1)) 

			fixedTime -= stepTime

			updatePosition := func(g *GameObject) {
				if g.Physics != nil {

					b := g.Physics.Body
					r := g.Transform().WorldRotation()
					g.Transform().SetWorldRotation(NewVector3(r.X, r.Y, 180-(float32(b.Angle())*DegreeConst)))
					pos := b.Position()
					g.Transform().SetWorldPosition(NewVector2(float32(pos.X), float32(pos.Y)))

					//fmt.Println(r.Z, b.Transform.Angle())

				}
			}

			Iter(arr, updatePosition)

			for _, i := range Space.Arbiters {
				if i.NumContacts == 0 {
					continue
				}
				a, _ := i.ShapeA.Body.UserData.(*Physics)
				b, _ := i.ShapeB.Body.UserData.(*Physics)
				if a != nil && b != nil {
					*a.currentCollision = *i
					*b.currentCollision = *i
					if (a.lastCollision.ShapeA == a.currentCollision.ShapeA && a.lastCollision.ShapeB == a.currentCollision.ShapeB) ||
						(a.lastCollision.ShapeA == a.currentCollision.ShapeB && a.lastCollision.ShapeB == a.currentCollision.ShapeA) {
						onCollisionGameObject(a.GameObject(), a.currentCollision)
					} else {
						if a.lastCollision.ShapeA != nil && a.lastCollision.ShapeB != nil {
							onCollisionExitGameObject(a.GameObject(), a.lastCollision)
						}
						onCollisionEnterGameObject(a.GameObject(), a.currentCollision)
						onCollisionGameObject(a.GameObject(), a.currentCollision)
					}
					if (b.lastCollision.ShapeA == b.currentCollision.ShapeA && b.lastCollision.ShapeB == b.currentCollision.ShapeB) ||
						(b.lastCollision.ShapeA == b.currentCollision.ShapeB && b.lastCollision.ShapeB == b.currentCollision.ShapeA) {
						onCollisionGameObject(b.GameObject(), b.currentCollision)
					} else {
						if b.lastCollision.ShapeA != nil && b.lastCollision.ShapeB != nil {
							onCollisionExitGameObject(b.GameObject(), b.lastCollision)
						}
						onCollisionEnterGameObject(b.GameObject(), b.currentCollision)
						onCollisionGameObject(b.GameObject(), b.currentCollision)
					}

				}
			}

			for _, b := range Space.AllBodies {
				g, ok := b.UserData.(*Physics)
				if ok && g != nil {
					if g.lastCollision.ShapeA != nil && g.lastCollision.ShapeB != nil &&
						g.currentCollision.ShapeA == nil && g.currentCollision.ShapeB == nil {
						onCollisionExitGameObject(g.GameObject(), g.lastCollision)
					}
				}
			}

			//}

			//time.Sleep(time.Millisecond * 20)

		}
		physicsDelta := timer.StopCustom("Physics time")

		timer.StartCustom("Update routines")
		Iter(arr, udpateGameObject)
		updateDelta := timer.StopCustom("Update routines")

		timer.StartCustom("LateUpdate routines")
		Iter(arr, lateudpateGameObject)
		lateUpdateDelta := timer.StopCustom("LateUpdate routines")

		timer.StartCustom("Draw routines")
		Iter(arr, drawGameObject)
		drawDelta := timer.StopCustom("Draw routines")

		timer.StartCustom("coroutines")
		RunCoroutines()
		coroutinesDelta := timer.StopCustom("coroutines")

		stepDelta := timer.Stop()

		if Debug {
			fmt.Println()
			fmt.Println("##################")
			if InternalFPS < 20 {
				fmt.Println("FPS is lower than 20. FPS:", InternalFPS)
			} else if InternalFPS < 30 {
				fmt.Println("FPS is lower than 30. FPS:", InternalFPS)
			} else if InternalFPS < 40 {
				fmt.Println("FPS is lower than 40. FPS:", InternalFPS)
			}
			if stepDelta > 17*time.Millisecond {
				fmt.Println("StepDelta time is lower than normal")
			}
			fmt.Println("Debugging Times:")
			fmt.Println("Step time", stepDelta)
			fmt.Println("Destroy time", destroyDelta)
			fmt.Println("Start time", startDelta)
			fmt.Println("FixedUpdate time", fixedUpdateDelta)
			fmt.Println("Physics time", physicsDelta)
			fmt.Println("Update time", updateDelta)
			fmt.Println("LateUpdate time", lateUpdateDelta)
			fmt.Println("Draw time", drawDelta)
			fmt.Println("Coroutines time", coroutinesDelta)
			fmt.Println("##################")
			fmt.Println()
		}

		UpdateInput()
	}
	glfw.SwapBuffers()
	now := time.Now()
	deltaTime = float32(now.Sub(before).Nanoseconds()/int64(time.Millisecond)) / 1000
}

func Iter(objs []*GameObject, f func(*GameObject)) {
	l := len(objs)
	for i := l - 1; i >= 0; i-- {
		f(objs[i])
		arr2 := objs[i].Transform().Children()
		Iter2(arr2, f)
	}
}

func Iter2(objs []*Transform, f func(*GameObject)) {
	l := len(objs)
	for i := l - 1; i >= 0; i-- {
		f(objs[i].GameObject())
		arr2 := objs[i].Children()
		Iter2(arr2, f)
	}
}

func drawGameObject(gameObject *GameObject) {

	//mat := gameObject.Transform().Matrix()

	//gl.LoadMatrixf(mat.Ptr())

	l := len(gameObject.components)
	comps := gameObject.components

	for i := l - 1; i >= 0; i-- {
		comps[i].Draw()
	}
}

func IterExcept(objs []*GameObject, f func(*GameObject), except *GameObject) {
	l := len(objs)
	for i := l - 1; i >= 0; i-- {
		if objs[i] != except {
			f(objs[i])
		}
		arr2 := objs[i].Transform().Children()
		Iter2Except(arr2, f, except)
	}
}

func Iter2Except(objs []*Transform, f func(*GameObject), except *GameObject) {
	l := len(objs)
	for i := l - 1; i >= 0; i-- {
		if objs[i].GameObject() != except {
			f(objs[i].GameObject())
		}
		arr2 := objs[i].Children()
		Iter2Except(arr2, f, except)
	}
}

func startGameObject(gameObject *GameObject) {
	l := len(gameObject.components)
	comps := gameObject.components

	for i := l - 1; i >= 0; i-- {
		if !comps[i].started() {
			comps[i].setStarted(true)
			comps[i].Start()
		}
	}
}

func destoyGameObject(gameObject *GameObject) {
	if gameObject.destoryMark {
		gameObject.destroy()

	}
}

func onCollisionGameObject(gameObject *GameObject, arb *c.Arbiter) {
	l := len(gameObject.components)
	comps := gameObject.components

	for i := l - 1; i >= 0; i-- {
		comps[i].OnCollision(NewCollision(arb))
	}
}

func onCollisionEnterGameObject(gameObject *GameObject, arb *c.Arbiter) {
	l := len(gameObject.components)
	comps := gameObject.components

	for i := l - 1; i >= 0; i-- {
		comps[i].OnCollisionEnter(NewCollision(arb))
	}
}

func onCollisionExitGameObject(gameObject *GameObject, arb *c.Arbiter) {
	l := len(gameObject.components)
	comps := gameObject.components

	for i := l - 1; i >= 0; i-- {
		comps[i].OnCollisionExit(NewCollision(arb))
	}
}

func udpateGameObject(gameObject *GameObject) {
	l := len(gameObject.components)
	comps := gameObject.components

	for i := l - 1; i >= 0; i-- {
		comps[i].Update()
	}
}

func lateudpateGameObject(gameObject *GameObject) {
	l := len(gameObject.components)
	comps := gameObject.components

	for i := l - 1; i >= 0; i-- {
		comps[i].LateUpdate()
	}
}

func fixedUdpateGameObject(gameObject *GameObject) {
	l := len(gameObject.components)
	comps := gameObject.components

	for i := l - 1; i >= 0; i-- {
		comps[i].FixedUpdate()
	}
}

func initGL() (err error) {
	gl.Init()
	gl.ShadeModel(gl.SMOOTH)
	gl.ClearColor(0, 0, 0, 0)
	gl.ClearDepth(1)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Hint(gl.PERSPECTIVE_CORRECTION_HINT, gl.NICEST)
	gl.DepthFunc(gl.NEVER)
	gl.Enable(gl.BLEND)
	gl.DepthMask(true)

	loadShader()

	return
}

func drawQuad(srcwidth, destwidth, srcheight, destheight float32) {
	gl.Begin(gl.QUADS)
	gl.TexCoord2i(0, 0)
	gl.Vertex2f(-1, -1)
	gl.TexCoord2i(int(srcwidth), 0)
	gl.Vertex2f(-1+destwidth, -1)
	gl.TexCoord2i(int(srcwidth), int(srcheight))
	gl.Vertex2f(-1+destwidth, -1+destheight)
	gl.TexCoord2i(0, int(srcheight))
	gl.Vertex2f(-1, -1+destheight)
	gl.End()
}

func onResize(w, h int) {
	if h <= 0 {
		h = 1
	}
	if w <= 0 {
		w = 1
	}

	Height = h
	Width = w

	gl.Viewport(0, 0, w, h)

	if GetScene() != nil && GetScene().SceneBase().Camera != nil {
		GetScene().SceneBase().Camera.UpdateResolution()
	}
}
