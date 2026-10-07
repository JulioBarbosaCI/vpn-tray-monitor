// Package logging grava um log rotativo ao lado do executavel.
package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Limites do arquivo de log. Um programa que roda o dia todo nao pode
// crescer sem limite no disco do usuario.
const (
	DefaultMaxSizeKB = 512
	timestampLayout  = "2006-01-02 15:04:05"
	backupSuffix     = ".1"
	filePerm         = 0o600
)

// Logger grava mensagens com data/hora em arquivo, rotacionando por tamanho.
type Logger struct {
	mu        sync.Mutex
	path      string
	file      *os.File
	maxBytes  int64
	alsoStdio io.Writer
}

// New abre (ou cria) o arquivo de log no caminho informado.
func New(path string, maxSizeKB int, alsoStdio io.Writer) (*Logger, error) {
	if maxSizeKB <= 0 {
		maxSizeKB = DefaultMaxSizeKB
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("criando a pasta do log: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, filePerm)
	if err != nil {
		return nil, fmt.Errorf("abrindo o log %s: %w", path, err)
	}
	return &Logger{
		path:      path,
		file:      file,
		maxBytes:  int64(maxSizeKB) * 1024,
		alsoStdio: alsoStdio,
	}, nil
}

// Path devolve o caminho do arquivo, usado pelo menu "Abrir log".
func (l *Logger) Path() string { return l.path }

// Close libera o arquivo.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func (l *Logger) Infof(format string, args ...any)  { l.write("INFO ", format, args...) }
func (l *Logger) Warnf(format string, args ...any)  { l.write("AVISO", format, args...) }
func (l *Logger) Errorf(format string, args ...any) { l.write("ERRO ", format, args...) }

// write monta a linha e a grava, rotacionando antes se o arquivo estourou.
func (l *Logger) write(level, format string, args ...any) {
	line := fmt.Sprintf("%s [%s] %s\n",
		time.Now().Format(timestampLayout), level, fmt.Sprintf(format, args...))

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.alsoStdio != nil {
		_, _ = io.WriteString(l.alsoStdio, line)
	}
	if l.file == nil {
		return
	}
	l.rotateIfNeededLocked(int64(len(line)))
	// Um log que falha nao pode derrubar o monitor: a falha e ignorada de
	// proposito, ja que nao ha outro canal para reporta-la.
	_, _ = l.file.WriteString(line)
}

// rotateIfNeededLocked troca o arquivo por um .1 quando o limite e atingido.
// Guarda apenas uma geracao anterior: o suficiente para investigar uma queda
// recente sem ocupar espaco indefinidamente.
func (l *Logger) rotateIfNeededLocked(incoming int64) {
	info, err := l.file.Stat()
	if err != nil || info.Size()+incoming <= l.maxBytes {
		return
	}
	if err := l.file.Close(); err != nil {
		return
	}
	// A falha ao rotacionar nao deve impedir a reabertura do log.
	_ = os.Rename(l.path, l.path+backupSuffix)

	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, filePerm)
	if err != nil {
		l.file = nil
		return
	}
	l.file = file
}
