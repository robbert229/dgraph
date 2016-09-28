package main

// #include <stdint.h>
// #include <stdlib.h>
// #cgo CFLAGS: -I/usr/local/include -DU_DISABLE_RENAMING=1
// #cgo LDFLAGS: -L/usr/local/lib -licuuc -licui18n -licuio -licudata -licule -liculx
// #include <unicode/ustring.h>
// #include <unicode/ubrk.h>
// typedef int MyType;
// int fortytwo(MyType* t) { return *t + 1; }
//
// UBreakIterator* MyFunc(const char* s, int l, int* error) {
//     UErrorCode     status = U_ZERO_ERROR;
//     UBreakIterator* bi = ubrk_open(UBRK_WORD, "", (const UChar*)s, l, &status);
//			 *error = status;
//     return bi;
// }
//
//
import "C"

import "fmt"

//import "unsafe"

type BreakIterator struct {
	c *C.UBreakIterator
}

func OpenBreakIterator(s string) *BreakIterator {
	var err C.int
	c := C.MyFunc(C.CString(s), C.int(len(s)), &err)
	fmt.Println(err)
	return &BreakIterator{c}
}

func main() {
	OpenBreakIterator("在香港分享有60    多人來參加")
	//OpenBreakIterator("hello")
	//s := "在香港分享有60    多人來參加"
	//var status int
	//	C.ubrk_open(C.UBRK_WORD, "en_us", s, len(s), &status)
	//C.ubrk_open(1, C.CString("en_us"), C.CString(s), C.int(len(s)), &status)
	//var x int32
	//fmt.Println(C.fortytwo(&x))
}
