package main

import (
	"flag"
	"reflect"
	"testing"
)

func newConductorSetupFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("conductor setup", flag.ContinueOnError)
	fs.Bool("json", false, "")
	fs.Bool("no-heartbeat", false, "")
	fs.Bool("heartbeat", false, "")
	fs.String("description", "", "")
	fs.String("shared-claude-md", "", "")
	fs.String("claude-md", "", "")
	return fs
}

func TestParseConductorSetupArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantName   string
		wantExtras []string
		wantDesc   string
		wantJSON   bool
		wantNoHB   bool
		wantHasErr bool
	}{
		{
			name:     "name before string flag",
			args:     []string{"ops", "--description", "Ops monitor"},
			wantName: "ops",
			wantDesc: "Ops monitor",
		},
		{
			name:     "string flag before name",
			args:     []string{"--description", "Ops monitor", "ops"},
			wantName: "ops",
			wantDesc: "Ops monitor",
		},
		{
			name:     "bool flags and name",
			args:     []string{"--json", "--no-heartbeat", "ops"},
			wantName: "ops",
			wantJSON: true,
			wantNoHB: true,
		},
		{
			name:       "extra positional args",
			args:       []string{"ops", "--description", "Ops monitor", "extra"},
			wantName:   "ops",
			wantExtras: []string{"extra"},
			wantDesc:   "Ops monitor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := newConductorSetupFlagSet()
			gotName, gotExtras, err := parseConductorSetupArgs(fs, tt.args)
			if (err != nil) != tt.wantHasErr {
				t.Fatalf("err = %v, wantHasErr = %v", err, tt.wantHasErr)
			}
			if gotName != tt.wantName {
				t.Fatalf("name = %q, want %q", gotName, tt.wantName)
			}
			if len(gotExtras) == 0 && len(tt.wantExtras) == 0 {
				// Treat nil and empty as equivalent for absent extra args.
			} else if !reflect.DeepEqual(gotExtras, tt.wantExtras) {
				t.Fatalf("extras = %v, want %v", gotExtras, tt.wantExtras)
			}

			desc := fs.Lookup("description").Value.String()
			if desc != tt.wantDesc {
				t.Fatalf("description = %q, want %q", desc, tt.wantDesc)
			}
			if fs.Lookup("json").Value.String() == "true" != tt.wantJSON {
				t.Fatalf("json = %v, want %v", fs.Lookup("json").Value.String() == "true", tt.wantJSON)
			}
			if fs.Lookup("no-heartbeat").Value.String() == "true" != tt.wantNoHB {
				t.Fatalf("no-heartbeat = %v, want %v", fs.Lookup("no-heartbeat").Value.String() == "true", tt.wantNoHB)
			}
		})
	}
}
