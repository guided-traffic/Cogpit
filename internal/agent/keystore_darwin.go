//go:build darwin && cgo

package agent

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>

// cogpitKeyQuery builds the query of one generic password: the service and the account, both
// UTF-8. The caller releases it. NULL on failure.
static CFMutableDictionaryRef cogpitKeyQuery(const char *service, const char *account) {
	CFStringRef svc = CFStringCreateWithCString(kCFAllocatorDefault, service, kCFStringEncodingUTF8);
	CFStringRef acc = CFStringCreateWithCString(kCFAllocatorDefault, account, kCFStringEncodingUTF8);
	if (svc == NULL || acc == NULL) {
		if (svc != NULL) CFRelease(svc);
		if (acc != NULL) CFRelease(acc);
		return NULL;
	}
	CFMutableDictionaryRef q = CFDictionaryCreateMutable(kCFAllocatorDefault, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	if (q != NULL) {
		CFDictionarySetValue(q, kSecClass, kSecClassGenericPassword);
		CFDictionarySetValue(q, kSecAttrService, svc);
		CFDictionarySetValue(q, kSecAttrAccount, acc);
	}
	CFRelease(svc);
	CFRelease(acc);
	return q;
}

// cogpitKeySet stores data under service and account: SecItemUpdate when the item exists, else
// SecItemAdd. The key never passes through a command line or another process.
static OSStatus cogpitKeySet(const char *service, const char *account, const void *data, int len) {
	CFMutableDictionaryRef q = cogpitKeyQuery(service, account);
	if (q == NULL) {
		return errSecAllocate;
	}
	CFDataRef value = CFDataCreate(kCFAllocatorDefault, (const UInt8 *)data, (CFIndex)len);
	if (value == NULL) {
		CFRelease(q);
		return errSecAllocate;
	}
	CFMutableDictionaryRef attrs = CFDictionaryCreateMutable(kCFAllocatorDefault, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	if (attrs == NULL) {
		CFRelease(value);
		CFRelease(q);
		return errSecAllocate;
	}
	CFDictionarySetValue(attrs, kSecValueData, value);
	OSStatus st = SecItemUpdate(q, attrs);
	if (st == errSecItemNotFound) {
		CFDictionarySetValue(q, kSecValueData, value);
		CFDictionarySetValue(q, kSecAttrLabel, CFSTR("Cogpit LLM connection"));
		st = SecItemAdd(q, NULL);
	}
	CFRelease(attrs);
	CFRelease(value);
	CFRelease(q);
	return st;
}

// cogpitKeyGet copies the stored data into a buffer the caller frees (and clears first).
static OSStatus cogpitKeyGet(const char *service, const char *account, void **out, int *outLen) {
	*out = NULL;
	*outLen = 0;
	CFMutableDictionaryRef q = cogpitKeyQuery(service, account);
	if (q == NULL) {
		return errSecAllocate;
	}
	CFDictionarySetValue(q, kSecReturnData, kCFBooleanTrue);
	CFDictionarySetValue(q, kSecMatchLimit, kSecMatchLimitOne);
	CFTypeRef result = NULL;
	OSStatus st = SecItemCopyMatching(q, &result);
	CFRelease(q);
	if (st != errSecSuccess) {
		return st;
	}
	if (result == NULL || CFGetTypeID(result) != CFDataGetTypeID()) {
		if (result != NULL) CFRelease(result);
		return errSecDecode;
	}
	CFIndex n = CFDataGetLength((CFDataRef)result);
	if (n > 0) {
		void *buf = malloc((size_t)n);
		if (buf == NULL) {
			CFRelease(result);
			return errSecAllocate;
		}
		memcpy(buf, CFDataGetBytePtr((CFDataRef)result), (size_t)n);
		*out = buf;
		*outLen = (int)n;
	}
	CFRelease(result);
	return errSecSuccess;
}

// cogpitKeyDelete removes the item.
static OSStatus cogpitKeyDelete(const char *service, const char *account) {
	CFMutableDictionaryRef q = cogpitKeyQuery(service, account);
	if (q == NULL) {
		return errSecAllocate;
	}
	OSStatus st = SecItemDelete(q);
	CFRelease(q);
	return st;
}

// cogpitClear overwrites a buffer before it is freed.
static void cogpitClear(void *p, int n) {
	if (p != NULL && n > 0) {
		volatile unsigned char *b = (volatile unsigned char *)p;
		while (n-- > 0) *b++ = 0;
	}
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// errSecItemNotFound is the Security framework's "the item cannot be found".
const errSecItemNotFound = -25300

// macKeyStore keeps the keys as generic passwords in the user's login keychain, through the
// Security framework (SecItemAdd, SecItemCopyMatching, SecItemUpdate, SecItemDelete): never
// through the security command line, which would put a key on a command line every process can
// read.
type macKeyStore struct{ service string }

// NewSystemKeyStore returns the operating system's keychain: on macOS the login keychain under
// KeyService.
func NewSystemKeyStore() KeyStore { return macKeyStore{service: KeyService} }

func (k macKeyStore) Get(account string) (string, error) {
	svc, acc := C.CString(k.service), C.CString(account)
	defer C.free(unsafe.Pointer(svc))
	defer C.free(unsafe.Pointer(acc))
	var out unsafe.Pointer
	var n C.int
	st := C.cogpitKeyGet(svc, acc, &out, &n)
	if st == errSecItemNotFound {
		return "", ErrNoKey
	}
	if st != C.errSecSuccess {
		return "", fmt.Errorf("keychain: reading the API key failed (OSStatus %d)", int(st))
	}
	if out == nil {
		return "", nil
	}
	key := C.GoStringN((*C.char)(out), n)
	C.cogpitClear(out, n)
	C.free(out)
	return key, nil
}

func (k macKeyStore) Set(account, key string) error {
	svc, acc := C.CString(k.service), C.CString(account)
	defer C.free(unsafe.Pointer(svc))
	defer C.free(unsafe.Pointer(acc))
	n := C.int(len(key))
	data := C.CBytes([]byte(key))
	defer func() {
		C.cogpitClear(data, n)
		C.free(data)
	}()
	if st := C.cogpitKeySet(svc, acc, data, n); st != C.errSecSuccess {
		return fmt.Errorf("keychain: storing the API key failed (OSStatus %d)", int(st))
	}
	return nil
}

func (k macKeyStore) Delete(account string) error {
	svc, acc := C.CString(k.service), C.CString(account)
	defer C.free(unsafe.Pointer(svc))
	defer C.free(unsafe.Pointer(acc))
	st := C.cogpitKeyDelete(svc, acc)
	if st == errSecItemNotFound || st == C.errSecSuccess {
		return nil
	}
	return fmt.Errorf("keychain: removing the API key failed (OSStatus %d)", int(st))
}
