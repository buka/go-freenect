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

// freenect project freenect.go
package freenect


/*
#cgo LDFLAGS:	-lfreenect

#include <libfreenect/libfreenect.h>

void registerLogCallback(freenect_context* ctx);
void registerVideoCallback(freenect_device* dev);
void registerDepthCallback(freenect_device* dev);

*/
import "C"

import (
	"fmt"
	"unsafe"
)

type LoggerLevel	int
type VideoFormat	int32
type DepthFormat 	int32
type Resolution		int32
type LEDOption		int

const (
	LogFatal 					= LoggerLevel(C.FREENECT_LOG_FATAL)
	LogError					= LoggerLevel(C.FREENECT_LOG_ERROR)
	LogWarning				= LoggerLevel(C.FREENECT_LOG_WARNING)
	LogNotice					= LoggerLevel(C.FREENECT_LOG_NOTICE)
	LogInfo						= LoggerLevel(C.FREENECT_LOG_INFO)
	LogDebug					= LoggerLevel(C.FREENECT_LOG_DEBUG)
	LogSpew						= LoggerLevel(C.FREENECT_LOG_SPEW)
	LogFlood					= LoggerLevel(C.FREENECT_LOG_FLOOD)
)

const (
	LOW								= Resolution(C.FREENECT_RESOLUTION_LOW)
	MEDIUM						= Resolution(C.FREENECT_RESOLUTION_MEDIUM)
	HIGH							= Resolution(C.FREENECT_RESOLUTION_HIGH)
)

const (
	RGB								= VideoFormat(C.FREENECT_VIDEO_RGB)
	BAYER							= VideoFormat(C.FREENECT_VIDEO_BAYER)
	IR_8BIT						= VideoFormat(C.FREENECT_VIDEO_IR_8BIT)
	IR_10BIT					= VideoFormat(C.FREENECT_VIDEO_IR_10BIT)
	IR_10BIT_PACKED 	= VideoFormat(C.FREENECT_VIDEO_IR_10BIT_PACKED)
	YUV_RGB						= VideoFormat(C.FREENECT_VIDEO_YUV_RGB)
	YUV_RAW						= VideoFormat(C.FREENECT_VIDEO_YUV_RAW)
)

const (
	D11BIT						= DepthFormat(C.FREENECT_DEPTH_11BIT)
	D10BIT	        	= DepthFormat(C.FREENECT_DEPTH_10BIT)
	D11BIT_PACKED			= DepthFormat(C.FREENECT_DEPTH_11BIT_PACKED)
	D10BIT_PACKED			= DepthFormat(C.FREENECT_DEPTH_10BIT_PACKED)
	REGISTERED  			= DepthFormat(C.FREENECT_DEPTH_REGISTERED)
	MM 								= DepthFormat(C.FREENECT_DEPTH_MM)
)

const (
	OFF								= LEDOption(C.LED_OFF)
	GREEN							= LEDOption(C.LED_GREEN)
	RED								= LEDOption(C.LED_RED)
	YELLOW						= LEDOption(C.LED_YELLOW)
	BLINK_GREEN				= LEDOption(C.LED_BLINK_GREEN)
	BLINK_RED_YELLOW	= LEDOption(C.LED_BLINK_RED_YELLOW)
)

const (
	TILT_STOPPED			= int(C.TILT_STATUS_STOPPED)
	TILT_AT_LIMIT			= int(C.TILT_STATUS_LIMIT)
	TILT_MOVING				= int(C.TILT_STATUS_MOVING)
)

type Logger func(level int, message string)
type Freenect struct {
	ctx 			*C.freenect_context
	logger		Logger
	Devices		[]Device
}

type Device struct {
	index			int
	freenect 	*Freenect
	dev				*C.freenect_device
	video 		*VideoCamera
	depth 		*DepthCamera
	tilt			*Tilt
}

type Tilt struct {
	device		*Device
	Angle			float32
	Status		int
	AccelX		float32
	AccelY		float32
	AccelZ		float32
}

// required for the logging callback
var _freenect *Freenect = nil

func Initialize() (*Freenect, int) {
	if _freenect != nil {
		return nil, -999
	}

	var ctx *C.freenect_context
	rc := int(C.freenect_init(&ctx, nil))

	if rc != 0 {
		return nil, rc
	}

	C.freenect_select_subdevices(ctx, (C.freenect_device_flags)(C.FREENECT_DEVICE_MOTOR | C.FREENECT_DEVICE_CAMERA))

	d := int(C.freenect_num_devices(ctx))
	_freenect = &Freenect{ctx, nil, make([]Device, d)}

	for x := 0; x < d; x++ {
		_freenect.Devices[x].index = x
		_freenect.Devices[x].freenect = _freenect
	}

	go func() {
		var to C.struct_timeval
		to.tv_sec = 0
		to.tv_usec = 0
		rc := C.int(0)
		for _freenect != nil && rc >= 0 {
			rc = C.freenect_process_events_timeout(_freenect.ctx, &to)
		}
	}()

	return _freenect, 0
}

func (freenect *Freenect) Shutdown() int {
	_freenect = nil
	return int(C.freenect_shutdown(freenect.ctx))
}

func (freenect *Freenect) Log(logger Logger) {
	freenect.logger = logger
}

func (freenect *Freenect) LogLevel(level LoggerLevel) {
	C.registerLogCallback(freenect.ctx)
	C.freenect_set_log_level(freenect.ctx, C.freenect_loglevel(level))
}

//export logCallback
func logCallback(ctx unsafe.Pointer, level C.freenect_loglevel, msg *C.char) {
	if _freenect != nil && _freenect.logger != nil {
		_freenect.logger(int(level), C.GoString(msg))
	}
}


// Devices...

func (device *Device) Open() int {
	rc := int(C.freenect_open_device(device.freenect.ctx, &device.dev, C.int(device.index)))
	if rc == 0 {
		C.freenect_set_user(device.dev, unsafe.Pointer(device))
	}
	return rc
}

func (device *Device) Close() int {
	C.freenect_set_user(device.dev, nil)
	return int(C.freenect_close_device(device.dev))
}

func (device *Device) LED(option LEDOption) int {
	return int(C.freenect_set_led(device.dev, C.freenect_led_options(option)))
}

// this function will always refresh the tilt state info from the device
func (device *Device) GetTilt() *Tilt {
	if device.tilt == nil {
		device.tilt = &Tilt{device, 0, 0, 0, 0, 0}
	}

	device.tilt.Refresh()
	return device.tilt
}

func (tilt *Tilt) Refresh() {
	C.freenect_update_tilt_state(tilt.device.dev)
	state := C.freenect_get_tilt_state(tilt.device.dev)

	tilt.Angle = float32(C.freenect_get_tilt_degs(state))
	tilt.Status = int(C.freenect_get_tilt_status(state))
	var x, y, z C.double
	C.freenect_get_mks_accel(state, &x, &y, &z)
	tilt.AccelX = float32(x)
	tilt.AccelY = float32(y)
	tilt.AccelZ = float32(z)
}

func (tilt *Tilt) SetAngle(deg float64) int {
	return int(C.freenect_set_tilt_degs(tilt.device.dev, C.double(deg)))
}

type VideoSource 	func(bytes int) []byte
type VideoSink		func(buffer []byte, stamp int32)
type VideoCamera struct {
	device  *Device
	on 			bool
	bytes 	int
	source 	VideoSource
	sink		VideoSink
	current []byte
}

type DepthSource 	func(bytes int) []uint16
type DepthSink		func(buffer []uint16, stamp int32)
type DepthCamera struct {
	device  *Device
	on 			bool
	bytes 	int
	source  DepthSource
	sink		DepthSink
	current []uint16
}

func (device *Device) VideoCamera(res Resolution, fmt VideoFormat, source VideoSource, sink VideoSink) (*VideoCamera, int) {
	if source == nil || sink == nil {
		return nil, -998
	}

	mode := C.freenect_find_video_mode(C.freenect_resolution(res), C.freenect_video_format(fmt))
	if mode.is_valid == 0 {
		return nil, -999
	}

	rc := int(C.freenect_set_video_mode(device.dev, mode))
	if rc != 0 {
		return nil, rc
	}

	C.registerVideoCallback(device.dev)

	device.video = &VideoCamera{device, false, int(mode.bytes), source, sink, nil}
	return device.video, 0
}

func (device *Device) DepthCamera(res Resolution, fmt DepthFormat, source DepthSource, sink DepthSink) (*DepthCamera, int) {
	mode := C.freenect_find_depth_mode(C.freenect_resolution(res), C.freenect_depth_format(fmt))
	if mode.is_valid == 0 {
		return nil, -999
	}

	rc := int(C.freenect_set_depth_mode(device.dev, mode))
	if rc != 0 {
		return nil, rc
	}

	C.registerDepthCallback(device.dev)

	device.depth = &DepthCamera{device, false, int(mode.bytes), source, sink, nil}
	return device.depth, 0
}


func (camera *VideoCamera) Start() int {
	if camera.on == true {
		return 1
	}

	buffer := camera.source(camera.bytes)
	rc := int(C.freenect_set_video_buffer(camera.device.dev, unsafe.Pointer(&buffer[0])))
	if rc != 0 {
		fmt.Printf("Failed to set video buffer: %d\n", rc)
		return rc
	}
	camera.current = buffer

	rc = int(C.freenect_start_video(camera.device.dev))
	if rc != 0 {
		fmt.Printf("Failed to start video stream: %d\n", rc)
		return rc
	}

	fmt.Printf("Video stream started\n")
	camera.on = true
	return 0
}

func (camera *DepthCamera) Start() int {
	if camera.on == true {
		return 1
	}

	buffer := camera.source(camera.bytes)
	rc := int(C.freenect_set_depth_buffer(camera.device.dev, unsafe.Pointer(&buffer[0])))
	if rc != 0 {
		fmt.Printf("Failed to set depth buffer: %d\n", rc)
		return rc
	}
	camera.current = buffer

	rc = int(C.freenect_start_depth(camera.device.dev))
	if rc != 0 {
		fmt.Printf("Failed to start depth stream: %d\n", rc)
		return rc
	}

	fmt.Printf("Depth stream started\n")
	camera.on = true
	return 0
}

func (camera *VideoCamera) Stop() int {
	if camera.on == false {
		return 1
	}

	C.freenect_stop_video(camera.device.dev)
	fmt.Printf("Video stream stopped\n")
	return 0
}

func (camera *DepthCamera) Stop() int {
	if camera.on == false {
		return 1
	}

	C.freenect_stop_depth(camera.device.dev)
	fmt.Printf("Depth stream stopped\n")
	return 0
}

//export videoCallback
func videoCallback(dev unsafe.Pointer, frame unsafe.Pointer, timestamp C.uint32_t) {
	device := (*Device)(C.freenect_get_user((*C.freenect_device)(dev)))
	if device == nil || device.video == nil {
		panic("No video camera found")
	}

	camera := device.video

	if frame != unsafe.Pointer(&camera.current[0]) {
		panic("Unexpected video frame buffer pointer")
	}

	camera.sink(camera.current, int32(timestamp))

		// source can return nil to reuse same buffer
	buffer := camera.source(camera.bytes)
	if buffer != nil {
		rc := int(C.freenect_set_video_buffer(camera.device.dev, unsafe.Pointer(&buffer[0])))
		if rc != 0 {
			fmt.Printf("Failed to set video buffer: %d\n", rc)
			panic("Failed to set video buffer")
		}
		camera.current = buffer
	}
}

//export depthCallback
func depthCallback(dev unsafe.Pointer, frame unsafe.Pointer, timestamp C.uint32_t) {
	device := (*Device)(C.freenect_get_user((*C.freenect_device)(dev)))
	if device == nil || device.depth == nil {
		panic("No depth camera found")
	}

	camera := device.depth

	if frame != unsafe.Pointer(&camera.current[0]) {
		panic("Unexpected depth frame buffer pointer")
	}

	camera.sink(camera.current, int32(timestamp))

		// source can return nil to reuse same buffer
	buffer := camera.source(camera.bytes)
	if buffer != nil {
		rc := int(C.freenect_set_depth_buffer(camera.device.dev, unsafe.Pointer(&buffer[0])))
		if rc != 0 {
			fmt.Printf("Failed to set depth buffer: %d\n", rc)
			panic("Failed to set depth buffer")
		}
		camera.current = buffer
	}
}
