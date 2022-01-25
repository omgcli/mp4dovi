# mp4dovi

Change video codec id in MP4 files for Apple QuickTime conformance.

- Apple requires Dolby Vision file to have 'dvh1' as codec
  id. https://developer.apple.com/av-foundation/High-Dynamic-Range-Metadata-for-Apple-Devices.pdf
- Apple devices recognize 'dvhe' for
  HLS. https://developer.apple.com/documentation/http_live_streaming/hls_authoring_specification_for_apple_devices/hls_authoring_specification_for_apple_devices_appendixes

## Recommended codec id for Apple devices

| Avoid | Recommended |
|-------|-------------|
| dvhe  | dvh1        |
| hev1  | hvc1        |
| avc3  | avc1        |

## MP4 file specification
https://developer.apple.com/standards/qtff-2001.pdf
