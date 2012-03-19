Go-Freenect
===========
A Go language wrapper for libfreenect which is part of the [OpenKinect project](http://openkinect.org/wiki/Main_Page)

Current Status
--------------
Video (RGB, others untested) and depth acquistion working.  Motor and tilt work as well, please see test case for how to use Refresh().  FWIW, the tests are really more like samples at this point - I recognize this...

There is no formal notion of errors at this point.  Usually upon failure the value returned is coming directly from libfreenect.

Developed and tested on Linux (Mint, kernel 3.0.0-15-generic) x64

Getting Started
---------------
### Install libfreenect
It's easy enough to git and build. On OSX you can use homebrew.
On my system, it installed to /usr/local/lib64; double-check this is in your ld path.

### Go
This project is currently built against the Go 1 RC1 weekly.  This means, among other things, that you must have a GOPATH environment variable defined.

Compile and install go-freenect:

    go install

Run the tests:

    go test

### Troubleshooting
If you run into scary errors that look like this:

    libusb couldn't open USB device /dev/bus/usb/001/022: Permission denied.
    libusb requires write access to USB device nodes.
    LEVEL: 1  MSG: Could not open camera: -3

Then you need to change the permissions on the usb devices. The two methods are either manually chmodding or configuring a udev script.

If it looks like a test deadlocks, it probably did because there aren't enough threads for the go-routines.  Try this:

    export GOMAXPROCS=4

If a test panics or you ctrl-c out, you might leave the device in a wonky state.  Disconnect and reconnect it and all should be good.  Again pay attention to the permissions thing.

License
=======
[Apache 2.0](http://www.apache.org/licenses/LICENSE-2.0.html)


