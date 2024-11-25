# Changelog

## V1.1.6

### Fixed
- Summary view entry linking offset issue

## V1.1.5

### Added
- Confirmation on summary view submission to reduce unintended submits

### Fixed
- Different proj codes have same task id even though specified differently.
- Stop additional checks for linking proj code to scoro task when submitting with summary view.


## V1.1.4

### Fixed
- Login via env vars doesn't fetch activity list

## V1.1.3 

### Fixed 
- Logger passed to internal packages for logging
- Looping in Summary View when selecting tasks and acitivites
- Only check task when determining if upload should be skipped

## V1.1.2 (Current Release)

### Added
- Can skip uploading on specified proj codes. 
- Can select acitivity to upload with or skip
- Github runners to automate builds
