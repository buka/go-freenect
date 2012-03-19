/*
   Copyright 2011-2012 Garrick Evans

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package freenect_test

import (
	"fmt"
	"time"
	"hash/crc32"
	//"math"
	"os"
	"image"
	"image/color"
  "image/png"
	"testing"
	"freenect"
)

func TestOpenCloseLib(t *testing.T) {
	lib, rc := freenect.Initialize()
	if rc != 0 {
		t.Errorf("Initialize returned %d", rc)
	}
	lib.Shutdown()
}

func TestOpenCloseDevs(t *testing.T) {
	lib, rc := freenect.Initialize()

	var logger = func(level int, message string) {
		fmt.Printf("LEVEL: %d  MSG: %s", level, message)
	}

	if rc == 0 {
		defer lib.Shutdown()
		lib.Log(logger)
		lib.LogLevel(freenect.LogFlood)
		for i := 0; i < len(lib.Devices); i++ {
			dev := lib.Devices[i]
			rc = dev.Open()
			if rc != 0 {
				t.Errorf("Failed to open device %d. Returned %d", i, rc)
			}
			rc = dev.Close()
			if rc != 0 {
				t.Errorf("Failed to close device %d. Returned %d", i, rc)
			}
		}
	}
}

// uses a single video frame buffer that gets set on the first sourcing
// the processing occurs in the sink and future calls to source return nil
// which skips reseting the video buffer pointer
func TestVideoOneBufferNoSwap(t *testing.T) {
	t.Logf("TestVideoOneBufferNoSwap")

	var logger = func(level int, message string) {
		fmt.Printf("LEVEL: %d  MSG: %s", level, message)
	}

	var buffer []byte = nil
	var source = func(bytes int) []byte {
		if buffer == nil {
			fmt.Printf("Creating buffer\n")
			buffer = make([]byte, bytes)
			return buffer
		}

		return nil
	}

	var recvd = 0
	var first, last int64 = 0, 0
	var sink = func(frame []byte, stamp int32) {
		if &frame[0] != &buffer[0] {
			t.Errorf("Unknown frame buffer arrived")
		}

		if recvd == 0 {
			first = time.Now().UnixNano()
		}
		if recvd == 29 {
			last = time.Now().UnixNano()
		}

		crc := crc32.ChecksumIEEE(frame)
		fmt.Printf("Got frame %d stamped: %d crc32: %x\n", recvd, stamp, crc)
		recvd++
	}

	lib, rc := freenect.Initialize()
	if rc == 0 {
		defer lib.Shutdown()
		lib.Log(logger)
		lib.LogLevel(freenect.LogWarning)

		for i := 0; i < len(lib.Devices); i++ {
			dev := lib.Devices[i]
			rc = dev.Open()
			if rc != 0 {
				t.Errorf("Failed to open device. Returned %d", rc)
			}

			defer dev.Close()

			cam, rc := dev.VideoCamera(freenect.MEDIUM, freenect.RGB, source, sink)
			if rc != 0 {
				t.Errorf("No camera. Returned %d", rc)
			}

			cam.Start()
			defer cam.Stop()

			for recvd < 30 {fmt.Printf("")}
			fmt.Printf("Got 30 video frames in %d ms\n", (last - first)/1e6)
		}
	}
}

// uses a single video frame buffer that gets set on each sourcing
func TestVideoOneBufferWithSwap(t *testing.T) {
	t.Logf("TestVideoOneBufferWithSwap")

	var logger = func(level int, message string) {
		fmt.Printf("LEVEL: %d  MSG: %s", level, message)
	}

	var buffer []byte = nil
	var source = func(bytes int) []byte {
		if buffer == nil {
			fmt.Printf("Creating buffer\n")
			buffer = make([]byte, bytes)
		}

		return buffer
	}

	var recvd = 0
	var first, last int64 = 0, 0
	var sink = func(frame []byte, stamp int32) {
		if &frame[0] != &buffer[0] {
			t.Errorf("Unknown frame buffer arrived")
		}

		if recvd == 0 {
			first = time.Now().UnixNano()
		}
		if recvd == 29 {
			last = time.Now().UnixNano()
		}

		crc := crc32.ChecksumIEEE(frame)
		fmt.Printf("Got frame %d stamped: %d crc32: %x\n", recvd, stamp, crc)
		recvd++
	}

	lib, rc := freenect.Initialize()
	if rc == 0 {
		defer lib.Shutdown()
		lib.Log(logger)
		lib.LogLevel(freenect.LogWarning)

		for i := 0; i < len(lib.Devices); i++ {
			dev := lib.Devices[i]
			rc = dev.Open()
			if rc != 0 {
				t.Errorf("Failed to open device. Returned %d", rc)
			}

			defer dev.Close()

			cam, rc := dev.VideoCamera(freenect.MEDIUM, freenect.RGB, source, sink)
			if rc != 0 {
				t.Errorf("No camera. Returned %d", rc)
			}

			cam.Start()
			defer cam.Stop()

			for recvd < 30 {fmt.Printf("")}
			fmt.Printf("Got 30 video frames in %d ms\n", (last - first)/1e6)
		}
	}
}


// capture and process using two buffers
func TestVideoDoubleBuffer(t *testing.T) {
	t.Logf("TestVideoDoubleBuffer")

	var logger = func(level int, message string) {
		fmt.Printf("LEVEL: %d  MSG: %s", level, message)
	}

	buffers := make(chan []byte, 2)
	frames := make(chan []byte, 2)
	run := true
	recvd := 0

	go func(){
		fmt.Printf("Waiting to process frames\n")
		for run {
			select {
				// wait for the next frame
				case frame := <- frames:
					if recvd == 30 {
						return
					}

					// PNG each frame
					fname := fmt.Sprintf("vdbtest-%d.png",recvd)
					f, err := os.OpenFile(fname, os.O_CREATE | os.O_WRONLY, 0666)
	        if err == nil {
		        m := image.NewNRGBA(image.Rect(0,0, 640, 480))
	            for y := 0; y < 480; y++ {
			        for x := 0; x < 640; x++ {
	                	m.Set(x, y, color.NRGBA{uint8(frame[(y*640*3)+(3*x)]), uint8(frame[(y*640*3)+((3*x)+1)]), uint8(frame[(y*640*3)+((3*x)+2)]), 255})
	                }
	        	}
	        	png.Encode(f, m)
					}

					// return the buffer
					buffers <- frame
					recvd++

				default:
			}
		}
		fmt.Printf("Done processing frames\n")
	}()

	var once = true
	var source = func(bytes int) []byte {
		if once {
			once = false
			fmt.Printf("Allocating buffers\n")
			buffers <- make([]byte, bytes)
			buffers <- make([]byte, bytes)
			fmt.Printf("Allocated buffers\n")
		}
		return <- buffers
	}

	var sink = func(frame []byte, stamp int32) {
		frames <- frame
	}

	lib, rc := freenect.Initialize()
	if rc == 0 {
		defer lib.Shutdown()
		lib.Log(logger)
		lib.LogLevel(freenect.LogWarning)

		for i := 0; i < len(lib.Devices); i++ {
			dev := lib.Devices[i]
			rc = dev.Open()
			if rc != 0 {
				t.Errorf("Failed to open device. Returned %d", rc)
			}

			defer dev.Close()

			cam, rc := dev.VideoCamera(freenect.MEDIUM, freenect.RGB, source, sink)
			if rc != 0 {
				t.Errorf("No camera. Returned %d", rc)
			}

			cam.Start()
			defer cam.Stop()

			for recvd < 30 {fmt.Printf("")}
			run = false
		}
	}
}

func TestDepthOnly(t *testing.T) {
	t.Logf("TestDepthOnly")

	var logger = func(level int, message string) {
		fmt.Printf("LEVEL: %d  MSG: %s", level, message)
	}

	var buffer []uint16 = nil
	var source = func(bytes int) []uint16 {
		if buffer == nil {
			buffer = make([]uint16, bytes)
			return buffer
		}

		return nil
	}

	gamma := make([]float64, 2048)
	for i := 0; i <2048; i++ {
		v := float64(i)/2048.0
		v = v*v*v*6.0
		gamma[i] = v*6.0*256.0;
		//gamma[i] = 0.1236 * math.Tan((float64(i)/2842.5) + 1.1863)
	}
	fmt.Printf("%f %f\n",gamma[0],gamma[2047])

	var recvd = 0
	var sink = func(frame []uint16, stamp int32) {
		if &frame[0] != &buffer[0] {
			t.Errorf("Unknown depth buffer arrived")
		}

		// PNG each frame
		fname := fmt.Sprintf("depthtest-%d.png",recvd)
		f, err := os.OpenFile(fname, os.O_CREATE | os.O_WRONLY, 0666)
    if err == nil {
      m := image.NewNRGBA(image.Rect(0,0, 640, 480))
      for y := 0; y < 480; y++ {
      	for x := 0; x < 640; x++ {
      		d := buffer[x+(y*640)]
      		mm := uint16(gamma[d])
      		var rgba color.NRGBA

					lb := uint8(mm & 0xff)
					switch (mm>>8) {
						case 0:
							rgba = color.NRGBA{255, 255-lb, 255-lb, 255}
							break
						case 1:
							rgba = color.NRGBA{255, lb, 0, 255}
							break
						case 2:
							rgba = color.NRGBA{255-lb, 255, 0, 255}
							break
						case 3:
							rgba = color.NRGBA{0, 255, lb, 255}
							break
						case 4:
							rgba = color.NRGBA{0, 255-lb, 255, 255}
							break
						case 5:
							rgba = color.NRGBA{0, 0, 255-lb, 255}
							break
						default:
							rgba = color.NRGBA{0, 0, 0, 255}
							break;
					}
          m.Set(x, y, rgba)
        }
    	}
    	png.Encode(f, m)
		}

		fmt.Printf("Got frame %d stamped: %d\n", recvd, stamp)
		recvd++
	}

	lib, rc := freenect.Initialize()
	if rc == 0 {
		defer lib.Shutdown()
		lib.Log(logger)
		lib.LogLevel(freenect.LogWarning)

		for i := 0; i < len(lib.Devices); i++ {
			dev := lib.Devices[i]
			rc = dev.Open()
			if rc != 0 {
				t.Errorf("Failed to open device. Returned %d", rc)
			}

			defer dev.Close()

			cam, rc := dev.DepthCamera(freenect.MEDIUM, freenect.D11BIT, source, sink)
			if rc != 0 {
				t.Errorf("No camera. Returned %d", rc)
			}

			cam.Start()
			defer cam.Stop()

			for recvd < 30 {fmt.Printf("")}
		}
	}
}

func TestVideoAndDepth(t *testing.T) {
	t.Logf("TestVideoAndDepth")

	var logger = func(level int, message string) {
		fmt.Printf("LEVEL: %d  MSG: %s", level, message)
	}

	vframes := make(chan []byte)
	dframes := make(chan []uint16)
	run := 2
	ready := true
	recvd := 0

	gamma := make([]float64, 2048)
	for i := 0; i <2048; i++ {
		v := float64(i)/2048.0
		v = v*v*v*6.0
		gamma[i] = v*6.0*256.0;
	}

	go func(){
		fmt.Printf("Waiting to process frames\n")
		var video []byte = nil
		var depth []uint16 = nil

		var image = func(){
			fname := fmt.Sprintf("bothtest-%d.png",recvd)
			f, err := os.OpenFile(fname, os.O_CREATE | os.O_WRONLY, 0666)
      if err == nil {
        m := image.NewNRGBA(image.Rect(0,0, 1280, 480))
          for y := 0; y < 480; y++ {
	        	for x := 0; x < 640; x++ {
              m.Set(x, y, color.NRGBA{uint8(video[(y*640*3)+(3*x)]), uint8(video[(y*640*3)+((3*x)+1)]), uint8(video[(y*640*3)+((3*x)+2)]), 255})

		      		d := depth[x+(y*640)]
		      		mm := uint16(gamma[d])
		      		var rgba color.NRGBA

							lb := uint8(mm & 0xff)
							switch (mm>>8) {
								case 0:
									rgba = color.NRGBA{255, 255-lb, 255-lb, 255}
									break
								case 1:
									rgba = color.NRGBA{255, lb, 0, 255}
									break
								case 2:
									rgba = color.NRGBA{255-lb, 255, 0, 255}
									break
								case 3:
									rgba = color.NRGBA{0, 255, lb, 255}
									break
								case 4:
									rgba = color.NRGBA{0, 255-lb, 255, 255}
									break
								case 5:
									rgba = color.NRGBA{0, 0, 255-lb, 255}
									break
								default:
									rgba = color.NRGBA{0, 0, 0, 255}
									break;
							}
		          m.Set(x+640, y, rgba)
            }
      	}
      	png.Encode(f, m)
      }
      video = nil
      depth = nil
      recvd++
		}

		for run>0 {
			select {
				case frame := <- vframes:
					video = make ([]byte, len(frame))
					copy(video, frame)

					if depth != nil {
						image()
					}


				case frame := <- dframes:
					depth = make ([]uint16, len(frame))
					copy(depth, frame)

					if video != nil {
						image()
					}

				default:
			}
		}
		fmt.Printf("Done processing frames\n")
	}()

	var vbuffer []byte = nil
	var vsource = func(bytes int) []byte {
		if vbuffer == nil {
			vbuffer = make([]byte, bytes)
			return vbuffer
		}

		return nil
	}
	var vsink = func(frame []byte, stamp int32) {
		if ready {
			vframes <- frame
		} else {
			run--
		}
	}

	var dbuffer []uint16 = nil
	var dsource = func(bytes int) []uint16 {
		if dbuffer == nil {
			dbuffer = make([]uint16, bytes)
			return dbuffer
		}

		return nil
	}

	var dsink = func(frame []uint16, stamp int32) {
		if ready {
			dframes <- frame
		} else {
			run--
		}
	}

	lib, rc := freenect.Initialize()
	if rc == 0 {
		defer lib.Shutdown()
		lib.Log(logger)
		lib.LogLevel(freenect.LogWarning)

		for i := 0; i < len(lib.Devices); i++ {
			dev := lib.Devices[i]
			rc = dev.Open()
			if rc != 0 {
				t.Errorf("Failed to open device. Returned %d", rc)
			}

			defer dev.Close()

			vcam, rc := dev.VideoCamera(freenect.MEDIUM, freenect.RGB, vsource, vsink)
			if rc != 0 {
				t.Errorf("No video camera. Returned %d", rc)
			}

			dcam, rc := dev.DepthCamera(freenect.MEDIUM, freenect.D11BIT, dsource, dsink)
			if rc != 0 {
				t.Errorf("No depth camera. Returned %d", rc)
			}

			vcam.Start()
			defer vcam.Stop()
			dcam.Start()
			defer dcam.Stop()

			for recvd < 30 {fmt.Printf("")}
			ready = false
		}
	}
}

func TestTilt(t *testing.T) {
	lib, rc := freenect.Initialize()

	if rc == 0 {
		defer lib.Shutdown()

		for i := 0; i < len(lib.Devices); i++ {
			dev := lib.Devices[i]
			rc = dev.Open()
			if rc != 0 {
				t.Errorf("Failed to open device. Returned %d", rc)
			}

			defer dev.Close()


			dev.LED(freenect.OFF)
			tilt := dev.GetTilt()

			run := true
			go func(){
				for run { tilt.Refresh() }
			}()

			dump := func() {
				motor := ""
				switch (tilt.Status) {
					case freenect.TILT_STOPPED: motor = "stopped"; break;
					case freenect.TILT_AT_LIMIT: motor = "at limit"; break;
					default: motor = "moving"
				}
				fmt.Printf("Current settings -- angle %f degrees, motor %s, accel (%f, %f, %f)\n",tilt.Angle, motor, tilt.AccelX, tilt.AccelY, tilt.AccelZ)
			}

			dev.LED(freenect.BLINK_GREEN)
			dump()
			fmt.Printf("Leveling...\n")
			tilt.SetAngle(0.0)
			for tilt.Angle != 0.0 {
			}
			dump()
			time.Sleep(1e9)

			dev.LED(freenect.RED)
			fmt.Printf("27 degrees\n")
			tilt.SetAngle(27.0)
			for tilt.Status == freenect.TILT_MOVING {
			}
			dump()
			time.Sleep(1e9)

			dev.LED(freenect.YELLOW)
			fmt.Printf("-27 degrees\n")
			tilt.SetAngle(-27.0)
			for tilt.Status == freenect.TILT_MOVING {
			}
			dump()
			time.Sleep(1e9)

			fmt.Printf("Relevel\n")
			dev.LED(freenect.BLINK_RED_YELLOW)
			tilt.SetAngle(0.0)
			for tilt.Angle != 0.0 {
			}
			dump()
			time.Sleep(1e9)

			run = false
		}
	}
}

