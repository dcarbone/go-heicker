package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dcarbone/go-confinator"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/rs/zerolog"
)

const defaultConfigHCL = // language=hcl
`ip = "0.0.0.0"
port = 8191
max_size_mb = 2
max_concurrent = 10
serve_path = "/opt/go-heicker/public"`

type Config struct {
	IP            string `json:"ip" hcl:"ip"`
	Port          int    `json:"port" hcl:"port"`
	MaxSizeMB     int64  `json:"max_size_mb" hcl:"max_size_mb"`
	MaxConcurrent int    `json:"max_concurrent" hcl:"max_concurrent"`
	ServePath     string `json:"serve_path" hcl:"serve_path"`

	BuildInfo confinator.BuildInfo `json:"build_info"`
}

func (c *Config) MarshalZerologObject(ev *zerolog.Event) {
	ev.Str("ip", c.IP)
	ev.Int("port", c.Port)
	ev.Int64("max_size_mb", c.MaxSizeMB)
	ev.Int("max_concurrent", c.MaxConcurrent)
	ev.Str("serve_path", c.ServePath)

	ev.Interface("build_info", c.BuildInfo)
}

func buildConfig(fs *flag.FlagSet, bi confinator.BuildInfo) *Config {
	conf := new(Config)
	conf.BuildInfo = bi
	hclFile, hclDiags := hclsyntax.ParseConfig([]byte(defaultConfigHCL), "default.hcl", hcl.Pos{Line: 1, Column: 1})
	if hclDiags != nil && hclDiags.HasErrors() {
		writeDiagErrors("default.hcl", hclFile, hclDiags)
	}
	decodeFile(conf, false, "default.hcl", hclFile)
	addConfigFlags(conf, fs)
	return conf
}

func addConfigFlags(c *Config, fs *flag.FlagSet) {
	cf := confinator.NewConfinator()

	cf.FlagVar(fs, &c.IP, "ip", "IP to bind")
	cf.FlagVar(fs, &c.Port, "port", "Port to bind")
	cf.FlagVar(fs, &c.MaxSizeMB, "max-size-mb", "Maximum file upload size in MB")
	cf.FlagVar(fs, &c.MaxConcurrent, "max-concurrent", "Maximum number of allowable concurrent requests")
	cf.FlagVar(fs, &c.ServePath, "serve-path", "Serve filepath")
}

func writeDiagErrors(filename string, file *hcl.File, diags hcl.Diagnostics) {
	writer := hcl.NewDiagnosticTextWriter(os.Stdout, map[string]*hcl.File{filename: file}, 120, true)
	if err := writer.WriteDiagnostics(diags); err != nil {
		panic(fmt.Sprintf("error printing diagnostics: %v; Diagnostics: %v", err, diags))
	}
	panic("configuration parse error")
}

func decodeFile(base *Config, ignoreErrors bool, filename string, file *hcl.File) {
	if diags := gohcl.DecodeBody(file.Body, nil, base); diags.HasErrors() && !ignoreErrors {
		writeDiagErrors(filename, file, diags)
	}
}
