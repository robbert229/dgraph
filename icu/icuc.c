#include "icuc.h"

#define kMaxTokenSize 1000

struct Tokenizer {
	UChar* buf;  // Our input string converted to UChar array. We own this array.
	UBreakIterator* iter;
	int end;  // Index into buf. It tells us where the last token ends.
	char token[kMaxTokenSize];  // For storing results of TokenizerNext.
};

// NewTokenizer creates new Tokenizer object given some input string. Note we
// convert the input to an array of UChar where each UChar takes 2 bytes.
Tokenizer* NewTokenizer(const char* input, int len, UErrorCode* err) {
	Tokenizer* tokenizer = (Tokenizer*)malloc(sizeof(Tokenizer));
	tokenizer->buf = (UChar*)malloc(sizeof(UChar) * (len + 1));
	
	// Convert char array to UChar array.
	u_uastrncpy(tokenizer->buf, input, len);
	
	// Prepares our iterator object.
	tokenizer->iter = ubrk_open(UBRK_WORD, "", tokenizer->buf, len, err);
	tokenizer->end = ubrk_first(tokenizer->iter);
	return tokenizer;
}

// DestroyTokenizer frees Tokenizer object.
void DestroyTokenizer(Tokenizer* tokenizer) {
	ubrk_close(tokenizer->iter);
	free(tokenizer->buf);
	free(tokenizer);
}

// TokenizerNext copies the next token into tokenizer->token and returns
// tokenizer->token.
char* TokenizerNext(Tokenizer* tokenizer) {
  const int32_t new_end = ubrk_next(tokenizer->iter);
	// Want to copy tokenizer->end to new_end.
	u_austrncpy(tokenizer->token, tokenizer->buf + tokenizer->end,
		new_end - tokenizer->end);
	tokenizer->end = new_end;
	return tokenizer->token;
}

// TokenizerDone returns whether the tokenizer is out of tokens.
int TokenizerDone(Tokenizer* tokenizer) {
	return tokenizer->end == UBRK_DONE;
}
