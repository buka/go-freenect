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


#include <libfreenect/libfreenect.h>

void registerLogCallback(freenect_context* ctx) {
	extern void logCallback(freenect_context*, freenect_loglevel, const char*);
	freenect_set_log_callback(ctx, logCallback);
}

void registerVideoCallback(freenect_device* dev) {
	extern void videoCallback(freenect_device*, void*, uint32_t);
	freenect_set_video_callback(dev, videoCallback);
}

void registerDepthCallback(freenect_device* dev) {
	extern void depthCallback(freenect_device*, void*, uint32_t);
	freenect_set_depth_callback(dev, depthCallback);
}
