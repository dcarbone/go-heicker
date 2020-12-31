package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/dcarbone/go-confinator"
	"github.com/rs/zerolog"
)

var (
	BuildName   string
	BuildDate   string
	BuildBranch string
)

func main() {
	bi := confinator.NewBuildInfo(BuildName, BuildDate, BuildBranch, "0")

	fs := flag.NewFlagSet("go-heicker", flag.ContinueOnError)
	conf := buildConfig(fs, bi)
	if err := fs.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		panic(fmt.Sprintf("Error parsing flags: %v", err))
	}

	log := zerolog.New(
		zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = os.Stdout
			w.TimeFormat = time.RFC3339
		})).
		With().
		Timestamp().
		Str("product", "go-heicker").
		Logger()

	log.Info().Msg("go-heicker booting up")
	log.Info().Object("config", conf).Msg("Runtime config built")
	log.Info().Msg("Building webservice...")

	ws, err := newWebService(log, conf)
	if err != nil {
		log.Error().Err(err).Msg("Error building webservice")
		os.Exit(1)
	}

	errc := make(chan error, 1)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		errc <- ws.serve(conf.IP, conf.Port)
	}()

	select {
	case err := <-errc:
		if err != nil {
			log.Error().Err(err).Msg("Abnormal exit")
			os.Exit(1)
		}
		log.Warn().Msg("Listener closed")
	case sig := <-sigc:
		log.Warn().Str("signal", sig.String()).Msg("Exiting")
	}

	os.Exit(0)
}
