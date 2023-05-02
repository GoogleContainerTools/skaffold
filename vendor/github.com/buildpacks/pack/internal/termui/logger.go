package termui

import "io"

func (s *Termui) Debug(msg string) {
	// not implemented
}

func (s *Termui) Debugf(fmt string, v ...interface{}) {
	// not implemented
}

func (s *Termui) Info(msg string) {
	s.textChan <- msg
}

func (s *Termui) Infof(fmt string, v ...interface{}) {
	// not implemented
}

func (s *Termui) Warn(msg string) {
	// not implemented
}

func (s *Termui) Warnf(fmt string, v ...interface{}) {
	// not implemented
}

func (s *Termui) Error(msg string) {
	// not implemented
}

func (s *Termui) Errorf(fmt string, v ...interface{}) {
	// not implemented
}

func (s *Termui) Writer() io.Writer {
	// not implemented
	return nil
}

func (s *Termui) IsVerbose() bool {
	// not implemented
	return false
}
