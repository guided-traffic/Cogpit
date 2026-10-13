//go:build darwin && cgo

package locale

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>

// cogpitUTF8 copies a CFString into a C string the caller frees; NULL for none or one longer
// than 1 KiB (no pattern or locale identifier is).
static char *cogpitUTF8(CFStringRef s) {
	if (s == NULL) {
		return NULL;
	}
	CFIndex max = CFStringGetMaximumSizeForEncoding(CFStringGetLength(s), kCFStringEncodingUTF8) + 1;
	if (max <= 1 || max > 1024) {
		return NULL;
	}
	char *buf = malloc((size_t)max);
	if (buf == NULL) {
		return NULL;
	}
	if (!CFStringGetCString(s, buf, max, kCFStringEncodingUTF8)) {
		free(buf);
		return NULL;
	}
	return buf;
}

// cogpitPattern returns the current locale's pattern for a date style and a time style, the
// user's own formats from System Settings included; NULL on failure.
static char *cogpitPattern(CFDateFormatterStyle dateStyle, CFDateFormatterStyle timeStyle) {
	CFLocaleRef loc = CFLocaleCopyCurrent();
	if (loc == NULL) {
		return NULL;
	}
	CFDateFormatterRef f = CFDateFormatterCreate(kCFAllocatorDefault, loc, dateStyle, timeStyle);
	CFRelease(loc);
	if (f == NULL) {
		return NULL;
	}
	char *out = cogpitUTF8(CFDateFormatterGetFormat(f));
	CFRelease(f);
	return out;
}

// cogpitLocaleID returns the current locale's identifier, "en_DE"; NULL on failure.
static char *cogpitLocaleID(void) {
	CFLocaleRef loc = CFLocaleCopyCurrent();
	if (loc == NULL) {
		return NULL;
	}
	char *out = cogpitUTF8(CFLocaleGetIdentifier(loc));
	CFRelease(loc);
	return out;
}
*/
import "C"

import "unsafe"

// systemFormat reads the region's format from macOS (CoreFoundation's current locale, which is
// the region in System Settings, not $LANG: a Finder-started app has no $LANG): the short date
// style and the medium time style, as the Finder writes a date. A user's own formats and the
// 24-hour switch are part of those styles.
func systemFormat() (Format, bool) {
	date, timeOfDay, ok := systemPatterns()
	if !ok {
		return Format{}, false
	}
	id, _ := cString(C.cogpitLocaleID())
	src := "macOS region " + id + " (" + date + ", " + timeOfDay + ")"
	l, err := FromPatterns(date, timeOfDay)
	if err != nil {
		src += ": " + err.Error()
	}
	return Format{Layout: l, Source: src}, true
}

// systemPatterns returns macOS's patterns of the short date style and the medium time style.
func systemPatterns() (date, timeOfDay string, ok bool) {
	date, okDate := cString(C.cogpitPattern(C.kCFDateFormatterShortStyle, C.kCFDateFormatterNoStyle))
	timeOfDay, okTime := cString(C.cogpitPattern(C.kCFDateFormatterNoStyle, C.kCFDateFormatterMediumStyle))
	return date, timeOfDay, okDate && okTime
}

// cString takes a C string from the functions above and frees it.
func cString(c *C.char) (string, bool) {
	if c == nil {
		return "", false
	}
	defer C.free(unsafe.Pointer(c))
	return C.GoString(c), true
}
