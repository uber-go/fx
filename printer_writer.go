package fx

import "go.uber.org/zap/zapcore"

type writeSyncer struct{ p Printer }

// writeSyncerFromPrinter returns an implementation of zapcore.WriteSyncer
// used to support Logger option which implements Printer interface.
func writeSyncerFromPrinter(p Printer) zapcore.WriteSyncer {
	return &writeSyncer{p: p}
}

func (w *writeSyncer) Write(b []byte) (n int, err error) {
	w.p.Printf(string(b))
	return len(b), nil
}

func (w *writeSyncer) Sync() error {
	return nil
}
