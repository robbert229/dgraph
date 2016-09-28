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
// void MyFunc(const char* s, int l) {
//     UErrorCode     status = U_ZERO_ERROR;
//     ubrk_open(UBRK_WORD, "en_us", (const UChar*)s, l, &status);
// }
//
//
import "C"

//import "fmt"

//import "unsafe"

func main() {
	s := "hello"
	C.MyFunc(C.CString(s), C.int(len(s)))
	//s := "在香港分享有60    多人來參加"
	//var status int
	//	C.ubrk_open(C.UBRK_WORD, "en_us", s, len(s), &status)
	//C.ubrk_open(1, C.CString("en_us"), C.CString(s), C.int(len(s)), &status)
	//var x int32
	//fmt.Println(C.fortytwo(&x))
}
