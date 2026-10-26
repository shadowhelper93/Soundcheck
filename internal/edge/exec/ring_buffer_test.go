package exec

import (
	"testing"
)

func TestRingBuffer_Basic(t *testing.T) {
	rb := newRingBuffer(10)
	_, _ = rb.Write([]byte("hello"))
	if string(rb.Bytes()) != "hello" {
		t.Fatalf("expected 'hello', got %q", string(rb.Bytes()))
	}
}

func TestRingBuffer_Wrap(t *testing.T) {
	rb := newRingBuffer(10)
	_, _ = rb.Write([]byte("abcdefgh"))
	_, _ = rb.Write([]byte("ijkl"))
	expected := "cdefghijkl"
	if string(rb.Bytes()) != expected {
		t.Fatalf("expected %q, got %q", expected, string(rb.Bytes()))
	}
}

func TestRingBuffer_OversizedSingleWrite(t *testing.T) {
	rb := newRingBuffer(5)
	_, _ = rb.Write([]byte("0123456789"))
	expected := "56789"
	if string(rb.Bytes()) != expected {
		t.Fatalf("expected %q, got %q", expected, string(rb.Bytes()))
	}
}

func TestRingBuffer_ZeroCap(t *testing.T) {
	rb := newRingBuffer(0)
	_, _ = rb.Write([]byte("abc"))
	if rb.Bytes() != nil {
		t.Fatalf("expected nil for zero capacity")
	}
}

func TestRingBuffer_EmptyWrite(t *testing.T) {
	rb := newRingBuffer(5)
	_, _ = rb.Write([]byte{})
	if len(rb.Bytes()) != 0 {
		t.Fatalf("expected 0 bytes for empty write")
	}
}
