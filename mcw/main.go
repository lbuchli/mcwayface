package main

import (
	"time"

	"github.com/phoenixdevelops/go-wlroots/wlroots"
)

type Server struct {
	display wlroots.Display
	backend wlroots.Backend

	outputs []*Output
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

	// start the backend
	err := server.backend.Start()
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
		color:     [4]float32{1, 0, 0, 1},
	}

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

	// calculate a color based on the time difference from the last frame
	now := time.Now()
	delta := now.Sub(mcwOut.lastFrame)
	mcwOut.lastFrame = now
	deltaS := float32(delta.Seconds())

	for i := 0; i < 3; i++ {
		// get the index of the next color
		next := i + 1
		if next == 3 {
			next = 0
		}

		// if the next color is 0, increase
		if mcwOut.color[next] == 0 {
			mcwOut.color[i] += deltaS
			if mcwOut.color[i] >= 1 {
				mcwOut.color[next] = deltaS
				mcwOut.color[i] = 1
			}
		} else { // decrease
			mcwOut.color[i] -= deltaS
			if mcwOut.color[i] <= 0 {
				mcwOut.color[i] = 0
			}
		}
	}
	// end fancy color generation

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

	output.SwapBuffers()
	renderer.End()
}
