package engine

import (
	"testing"
)

func BenchmarkLockBarrier(b *testing.B) {
	var barrier Barrier = newLockBarrier()
	b.Run("Pause & Resume", func(bb *testing.B) {
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			barrier.Pause().Resume()
		}
	})

	b.Run("Wait", func(bb *testing.B) {
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			barrier.Wait()
		}
	})
}

func BenchmarkWaitGroupBarrier(b *testing.B) {
	var barrier Barrier = newWaitGroupBarrier()
	b.Run("Pause & Resume", func(bb *testing.B) {
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			barrier.Pause().Resume()
		}
	})

	b.Run("Wait", func(bb *testing.B) {
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			barrier.Wait()
		}
	})
}
