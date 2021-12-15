package main

import (
	"net/http"
	"os"
	"time"

	"bitbucket.org/stop-panic/signaling/internal/config"
	"bitbucket.org/stop-panic/signaling/internal/handler"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	loggingFormatJson = "json"
	loggingFormatText = "text"
)

func main() {
	conf, err := config.GetConfig()
	if err != nil {
		setLoggingFormat(loggingFormatText)
		setLoggingLevel(log.FatalLevel.String())
		log.Fatal("error while getting a config")
		return
	}

	setLoggingFormat(conf.Logs.Format)
	setLoggingLevel(conf.Logs.Level)

	upgrader := &websocket.Upgrader{
		HandshakeTimeout:  5 * time.Second,
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		Error:             nil,
		CheckOrigin:       nil,
		EnableCompression: true,
	}

	server := handler.NewServer(upgrader)

	sslEnable := isSslEnable(&conf.Server)

	if sslEnable {
		err = http.ListenAndServeTLS(conf.Server.Addr, conf.Server.TlsCert, conf.Server.TlsKey, server)
	}

	if !sslEnable {
		err = http.ListenAndServe(conf.Server.Addr, server)
	}

	if err != nil {
		log.WithError(err).Error("error while starting a server")
	}
}

func setLoggingFormat(format string) {
	switch format {
	case loggingFormatJson:
		log.SetFormatter(&log.JSONFormatter{})
	case loggingFormatText:
		log.SetFormatter(&log.TextFormatter{
			ForceColors:      true,
			QuoteEmptyFields: true,
		})
	default:
		log.Fatalf("invalid logging format: %s", format)
	}
}

func setLoggingLevel(level string) {
	lvl, err := log.ParseLevel(level)
	if err != nil {
		log.WithField("title", "invalid logging level").Fatal(err)
	}
	log.SetLevel(lvl)
	log.Infof("logging level set to: %s", lvl)
}

func isSslEnable(conf *config.Server) bool {
	if conf.TlsCert == "" || conf.TlsKey == "" {
		return false
	}

	return isFileExists(conf.TlsCert) && isFileExists(conf.TlsKey)
}

func isFileExists(filepath string) bool {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}