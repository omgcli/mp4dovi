# mp4dovi

**This fork works with 64bit box sizes in MP4 files. It can handle large(> 4GB) MP4 files without `faststart` option set.**

Change video codec in MP4 files for Apple QuickTime conformance. 

- Apple requires Dolby Vision file to have 'dvh1' as codec. https://developer.apple.com/av-foundation/High-Dynamic-Range-Metadata-for-Apple-Devices.pdf
- Apple devices recognize 'dvhe' for HLS though. https://developer.apple.com/documentation/http_live_streaming/hls_authoring_specification_for_apple_devices/hls_authoring_specification_for_apple_devices_appendixes

## Usage

```bash
usage: mp4dovi [options] files...
  -from string
      video codec to convert from (default "dvhe")
  -to string
      video codec to convert to (default "dvh1")
  -verbose
      enable verbose output

```

## Recommended codec id for Apple devices

| Avoid | Recommended |
|-------|-------------|
| dvhe  | dvh1        |
| hev1  | hvc1        |
| avc3  | avc1        |

## MP4 file specification
https://developer.apple.com/standards/qtff-2001.pdf
