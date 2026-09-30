package service

// Base62 alphabet: 0-9, A-Z, a-z (62 characters).
// The order matters: it determines the lexicographic properties of generated codes.
// Using this alphabet, Encode(1024) produces "gW".
const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// lookupTable provides O(1) decoding: maps a byte value to its index in the alphabet,
// or -1 if the byte is not a valid Base62 character.
// This avoids a linear scan of the alphabet string on every Decode call.
var lookupTable [256]int

func init() {
	// Initialize all entries to -1 (invalid character).
	for i := range lookupTable {
		lookupTable[i] = -1
	}
	// Set valid Base62 characters to their corresponding index.
	for i, ch := range alphabet {
		lookupTable[ch] = i
	}
}

// Encode converts a positive int64 to its Base62 string representation.
//
// Algorithm (repeated division):
//  1. Divide n by 62 → remainder r is the index of the next character in the alphabet.
//  2. Prepend alphabet[r] to the result string.
//  3. Set n = n / 62 and repeat until n == 0.
//
// Special case: n == 0 → "0" (the identity element).
//
// Example:
//
//	Encode(1024) → "GW"
//	  1024 / 62 = 16 remainder 32 → alphabet[32] = 'W'
//	    16 / 62 =  0 remainder 16 → alphabet[16] = 'G'
//	  Result (reversed): "GW"
func Encode(n int64) string {
	if n == 0 {
		return "0"
	}

	// Pre-allocate a small buffer. Most IDs in a short-URL system stay below
	// 62^6 ≈ 56 billion, so 10 bytes is more than enough.
	buf := make([]byte, 0, 10)

	for n > 0 {
		remainder := n % 62
		buf = append(buf, alphabet[remainder])
		n /= 62
	}

	// Reverse the buffer: the loop produces digits from least-significant to
	// most-significant, so we reverse to get the standard big-endian notation.
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}

	return string(buf)
}

// Decode converts a Base62 string back to its int64 representation.
// Returns -1 if the string is empty or contains a character outside the Base62 alphabet.
//
// Algorithm (positional evaluation — Horner's method):
//
//	result = 0
//	for each character c in s:
//	    result = result*62 + lookupTable[c]
//
// Example:
//
//	Decode("GW") → 1024
//	  'G' → index 16: result = 0*62 + 16 = 16
//	  'W' → index 32: result = 16*62 + 32 = 1024
func Decode(s string) int64 {
	if len(s) == 0 {
		return -1
	}

	var result int64
	for i := 0; i < len(s); i++ {
		idx := lookupTable[s[i]]
		if idx == -1 {
			return -1 // invalid character — not in Base62 alphabet
		}
		result = result*62 + int64(idx)
	}

	return result
}
