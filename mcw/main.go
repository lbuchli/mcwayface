package main

import (
	"fmt"
	"os"
	"time"

	"github.com/phoenixdevelops/go-wlroots/wlroots"
)

type Server struct {
	display wlroots.Display
	backend wlroots.Backend

	seat       wlroots.Seat
	compositor wlroots.Compositor
	xdgShell   wlroots.XDGShell

	outputs  []*Output
	surfaces []wlroots.XDGSurface
}

type Output struct {
	wlrOutput wlroots.Output
	lastFrame time.Time

	color [4]float32
}

func main() {
	server := new(Server)

	server.outputs = make([]*Output, 0)

	server.display = wlroots.NewDisplay()
	server.backend = wlroots.NewBackend(server.display)

	server.backend.OnNewOutput(server.newOuput)
	server.backend.Renderer().InitDisplay(server.display)

	// configure seat
	server.seat = wlroots.NewSeat(server.display, "seat0")

	server.compositor = wlroots.NewCompositor(server.display, server.backend.Renderer())

	server.xdgShell = wlroots.NewXDGShell(server.display)
	server.xdgShell.OnNewSurface(server.handleNewSurface)

	// setup socket for wayland clients to connect to
	socket, err := server.display.AddSocketAuto()
	if err != nil {
		panic(err)
	}

	// start the backend
	err = server.backend.Start()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Running compositor on wayland display '%s'\n", socket)
	err = os.Setenv("WAYLAND_DISPLAY", socket)
	if err != nil {
		panic(err)
	}

	// and run the display
	server.display.Run()
}

func (s *Server) newOuput(output wlroots.Output) {
	out := &Output{
		wlrOutput: output,
		lastFrame: time.Now(),
		color:     [4]float32{0, 0, 0, 1},
	}

	output.CreateGlobal()

	out.wlrOutput.OnDestroy(s.destroyOutput)
	out.wlrOutput.OnFrame(s.drawFrame)

	// set the output mode to the last mode (only if there are modes)
	// The last mode is usually the largest at the highest refresh rate
	modes := out.wlrOutput.Modes()
	if len(modes) > 0 {
		out.wlrOutput.SetMode(modes[len(modes)-1])
	}

	s.outputs = append(s.outputs, out)
}

func (s *Server) destroyOutput(output wlroots.Output) {
	for i, out := range s.outputs {
		if out.wlrOutput.Name() == output.Name() {
			// delete the output from the list
			s.outputs = append(s.outputs[:i], s.outputs[i+1:]...)
			break
		}
	}
}

func (s *Server) drawFrame(output wlroots.Output) {
	renderer := s.backend.Renderer()

	// search for our version of the output
	var mcwOut *Output
	for _, out := range s.outputs {
		if out.wlrOutput.Name() == output.Name() {
			mcwOut = out
			break
		}
	}
	// check if we haven't found it
	if mcwOut == nil {
		panic("Could not find display!")
	}

	width, height := output.EffectiveResolution()

	// try to make the current output the current OpenGL context
	_, err := output.MakeCurrent()
	if err != nil {
		panic("Could not change OpenGL context!")
	}

	renderer.Begin(output, width, height)
	renderer.Clear(&wlroots.Color{
		A: mcwOut.color[3],
		R: mcwOut.color[0],
		B: mcwOut.color[1],
		G: mcwOut.color[2],
	})

	for _, surface := range s.surfaces {

		surf := surface.Surface()
		state := surf.CurrentState()

		renderBox := &wlroots.Box{
			X:      20,
			Y:      20,
			Width:  state.Width(),
			Height: state.Height(),
		}

		matrix := &wlroots.Matrix{}
		transformMatrix := output.TransformMatrix()
		matrix.ProjectBox(renderBox, state.Transform(), 0, &transformMatrix)

		renderer.RenderTextureWithMatrix(surf.Texture(), matrix, 1)
		surf.SendFrameDone(time.Now())
	}

	output.SwapBuffers()
	renderer.End()
}

func (s *Server) handleNewSurface(surface wlroots.XDGSurface) {
	surface.OnDestroy(s.handleSurfaceDestroy)
	s.surfaces = append(s.surfaces, surface)
}

func (s *Server) handleSurfaceDestroy(surface wlroots.XDGSurface) {
	for i, sb := range s.surfaces {
		if surface == sb {
			s.surfaces = append(s.surfaces[:i], s.surfaces[i+1:]...)
		}
	}
}
