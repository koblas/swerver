package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	box "github.com/Delta456/box-cli-maker/v2"

	"github.com/jessevdk/go-flags"
	"github.com/koblas/swerver/pkg/handler"
	_ "gopkg.in/go-playground/validator.v9"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func loadConfig(path *string) handler.Configuration {
	if path != nil {
		config, _ := handler.LoadServeConfiguration(*path)
		return config
	}
	config, _ := handler.LoadServeConfiguration("swerver.json")
	return config
}

func main() {
	var opts struct {
		// Help          bool      `short:"h" long:"help" description:"Shows this help message"`
		Version       bool      `short:"v" long:"version" description:"Display the current version of serve"`
		Listen        []*string `short:"l" long:"listen" description:"Specify a URI endpoint on which to listen (see below) more than one may be specified to listen in multiple places" default:"5000"`
		Port          *string   `short:"p" long:"port" description:"Port (depreicated, use listen)" hidden:"true"`
		Debug         *bool     `short:"d" long:"debug" description:"Shows debugging information"`
		Single        *bool     `short:"s" long:"single" description:"Rewrite all not-found requests to 'index.html'"`
		NoClipboard   *bool     `short:"n" long:"no-clipboard" description:"Do not copy the local address to the clipboard"`
		NoCompression *bool     `short:"u" long:"no-compression" description:"Disable compression for files served"`
		Symlinks      *bool     `short:"S" long:"symlinks" description:"Resolve symlinks instead of showing 404 errors"`
		Config        *string   `short:"c" long:"config" description:"Specify custom path to 'serve.json'"`
	}

	args, err := flags.Parse(&opts)
	if err != nil {
		if !flags.WroteHelp(err) {
			panic(err)
		}
		os.Exit(0)
	}

	if opts.Version {
		fmt.Printf("0.1.0\n")
		os.Exit(0)
	}

	config := loadConfig(opts.Config)

	if opts.Single != nil {
		config.RenderSingle = *opts.Single
		config.Rewrites = append(config.Rewrites, handler.ConfigRewrite{
			Source:      "**",
			Destination: "/index.html",
		})
	}
	if opts.Debug != nil {
		config.Debug = *opts.Debug
	}
	if opts.NoClipboard != nil {
		config.Clipboard = !*opts.NoClipboard
	}
	if opts.NoCompression != nil {
		config.NoCompression = *opts.NoCompression
	}
	if opts.Port != nil {
		if len(opts.Listen) == 1 && *opts.Listen[0] == "5000" {
			opts.Listen = []*string{opts.Port}
		} else {
			opts.Listen = append(opts.Listen, opts.Port)
		}
	}
	if len(opts.Listen) == 0 {
		port := "5000"
		opts.Listen = []*string{&port}
	}
	if len(args) != 0 {
		config.Public = args[0]
	}
	if config.Public == "" {
		cwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		config.Public = cwd
	}

	/*
		fmt.Println("┌──────────────────────────────────────────────────┐")
		fmt.Println("│                                                  │")
		fmt.Println("│   Serving!                                       │")
		fmt.Println("│                                                  │")
		fmt.Println("│   - Local:            http://localhost:9000      │")
		fmt.Println("│   - On Your Network:  http://192.168.1.22:9000   │")
		fmt.Println("│                                                  │")
		if config.Clipboard {
			fmt.Println("│   Copied local address to clipboard!             │")
			fmt.Println("│                                                  │")
		}
		fmt.Println("└──────────────────────────────────────────────────┘")
	*/

	bx := box.New(box.Config{Px: 4, Py: 1})
	lines := []string{}

	for idx, item := range opts.Listen {
		lines = append(lines, fmt.Sprintf("- Local:       http://%s:%s", "localhost", *item))
		// lines = append(lines, fmt.Sprintf("%s    %s",
		// 	color.Magenta.Sprint("- Local"),
		// 	color.Info.Sprintf("http://%s:%s", "localhost", *item)))

		listener := func() {
			// mux := http.NewServeMux()
			// mux.Handle("/", handler.NewHandler(config))

			h := handler.NewHandler(config)

			router := chi.NewRouter()
			router.Use(middleware.Logger)
			if !config.NoCompression {
				router.Use(middleware.Compress(5))
			}

			h.AttachRoutes(router)

			server := http.Server{
				Addr:    fmt.Sprintf(":%s", *item),
				Handler: router,
			}

			if config.Ssl.KeyFile != "" && config.Ssl.CertFile != "" {
				log.Fatal(server.ListenAndServeTLS(config.Ssl.CertFile, config.Ssl.KeyFile))
			} else {
				log.Fatal(server.ListenAndServe())
			}
		}

		if idx == len(opts.Listen)-1 {
			bx.Println("Serving!", strings.Join(lines, "\n"))

			listener()
		} else {
			go listener()
		}
	}
}
