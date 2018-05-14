package structtags

import (
	"fmt"
	"runtime"
	"testing"
)

type testCopyAllPlatforms struct {
	DiagnosticsServerPath          string `yaml:"diagnosticsServerPath" copyTo:"-"`
	DiagnosticsServerNamedPipePath string `yaml:"-" copyTo:"DiagnosticsServerPath,GOOS=android,darwin,dragonfly,freebsd,!notARealOS,linux,nacl,netbsd,openbsd,plan9,solaris,windows"`
	DiagnosticsSocketPath          string `yaml:"-" copyTo:"DiagnosticsServerPath,GOOS=!android,!darwin,!dragonfly,!freebsd,notARealOS,!linux,!nacl,!netbsd,!openbsd,!plan9,!solaris,!windows"`
}

func (c testCopyAllPlatforms) getResult() string {
	return c.DiagnosticsServerPath
}
func (c testCopyAllPlatforms) getDesired() string {
	return c.DiagnosticsServerNamedPipePath
}

type testCopyNoPlatforms struct {
	DiagnosticsServerPath          string `copyTo:"-"`
	DiagnosticsServerNamedPipePath string `copyTo:"DiagnosticsServerPath,GOOS="`
	DiagnosticsSocketPath          string `copyTo:"DiagnosticsServerPath,GOOS=!android,!darwin,!dragonfly,!freebsd,notARealOS,!linux,!nacl,!netbsd,!openbsd,!plan9,!solaris,!windows"`
}

func (c testCopyNoPlatforms) getResult() string {
	return c.DiagnosticsServerPath
}
func (c testCopyNoPlatforms) getDesired() string {
	return c.DiagnosticsServerNamedPipePath
}

type testCopyMultipleTargets struct {
	DiagnosticsServerPath          string `copyTo:"-"`
	DiagnosticsServerPath2         string `copyTo:"-"`
	DiagnosticsServerNamedPipePath string `copyTo:"DiagnosticsServerPath,DiagnosticsServerPath2,GOOS="`
	DiagnosticsSocketPath          string `copyTo:"DiagnosticsServerPath,GOOS=!android,!darwin,!dragonfly,!freebsd,notARealOS,!linux,!nacl,!netbsd,!openbsd,!plan9,!solaris,!windows"`
}

func (c testCopyMultipleTargets) getResult() string {
	return fmt.Sprintf("%s:%s", c.DiagnosticsServerPath, c.DiagnosticsServerPath2)
}
func (c testCopyMultipleTargets) getDesired() string {
	return fmt.Sprintf("%s:%s", c.DiagnosticsServerNamedPipePath, c.DiagnosticsServerNamedPipePath)
}

type testIncompatibleTypes struct {
	DiagnosticsServerPath          int    `copyTo:"-"`
	DiagnosticsServerPath2         string `copyTo:"-"`
	DiagnosticsServerNamedPipePath string `copyTo:"DiagnosticsServerPath,DiagnosticsServerPath2,GOOS="`
	DiagnosticsSocketPath          string `copyTo:"DiagnosticsServerPath,GOOS=!android,!darwin,!dragonfly,!freebsd,notARealOS,!linux,!nacl,!netbsd,!openbsd,!plan9,!solaris,!windows"`
}

func (c testIncompatibleTypes) getResult() string {
	return fmt.Sprintf("%v", c.DiagnosticsServerPath)
}
func (c testIncompatibleTypes) getDesired() string {
	return fmt.Sprintf("2")
}

type nonExistentTarget struct {
	DiagnosticsServerPath          string `copyTo:"-"`
	DiagnosticsServerNamedPipePath string `copyTo:"DiagnosticsServerPath,DiagnosticsServerPath2,GOOS="`
	DiagnosticsSocketPath          string `copyTo:"DiagnosticsServerPath,GOOS=!android,!darwin,!dragonfly,!freebsd,notARealOS,!linux,!nacl,!netbsd,!openbsd,!plan9,!solaris,!windows"`
}

func (c nonExistentTarget) getResult() string {
	return c.DiagnosticsServerPath
}
func (c nonExistentTarget) getDesired() string {
	return c.DiagnosticsServerNamedPipePath
}

type testInput interface {
	getResult() string
	getDesired() string
}

func TestCopyTo(t *testing.T) {
	type args struct {
		s testInput
	}
	tests := []struct {
		name   string
		args   args
		wantEr bool
	}{
		{
			name: "test copy all platforms",
			args: args{
				s: &testCopyAllPlatforms{
					DiagnosticsServerNamedPipePath: "all platforms except the fake one",
					DiagnosticsSocketPath:          "no platforms except the fake one",
				},
			},
			wantEr: false,
		},
		{
			name: "test copy no platforms specified",
			args: args{
				s: &testCopyNoPlatforms{
					DiagnosticsServerNamedPipePath: "all platforms",
					DiagnosticsSocketPath:          "no platforms except the fake one",
				},
			},
			wantEr: false,
		},
		{
			name: "test copy to multiple targets",
			args: args{
				s: &testCopyMultipleTargets{
					DiagnosticsServerNamedPipePath: "all platforms",
					DiagnosticsSocketPath:          "no platforms except the fake one",
				},
			},
			wantEr: false,
		},
		{
			name: "test incompatible target type",
			args: args{
				s: &testIncompatibleTypes{
					DiagnosticsServerPath:          2,
					DiagnosticsServerNamedPipePath: "all platforms",
					DiagnosticsSocketPath:          "no platforms except the fake one",
				},
			},
			wantEr: true,
		},
		{
			name: "test nonexistent target",
			args: args{
				s: &nonExistentTarget{
					DiagnosticsServerNamedPipePath: "all platforms",
					DiagnosticsSocketPath:          "no platforms except the fake one",
				},
			},
			wantEr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CopyTo(tt.args.s); (err != nil) != tt.wantEr {
				t.Errorf("CopyTo() error = %v wanted = %v", err, tt.wantEr)
			} else {
				result := tt.args.s.(testInput).getResult()
				desired := tt.args.s.(testInput).getDesired()
				t.Logf("result: %v desired: %v", result, desired)
				if result != desired {
					t.Errorf(fmt.Sprintf("CopyTo() DiagnosticsServerPath != %s", desired))
				}
			}
		})
	}
}

func Test_isOSEligible(t *testing.T) {
	type args struct {
		OSString string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "IsEligible",
			args: args{
				OSString: runtime.GOOS,
			},
			want: true,
		},
		{
			name: "IsEligibleBlank",
			args: args{
				OSString: "",
			},
			want: true,
		},
		{
			name: "IsElligibleOtherOSExcluded",
			args: args{
				OSString: "!notARealOS",
			},
			want: true,
		},
		{
			name: "IsElligibleOtherOSExcludedExplicitlyIncluded",
			args: args{
				OSString: fmt.Sprintf("!notARealOS,%s", runtime.GOOS),
			},
			want: true,
		},
		{
			name: "NotEligible",
			args: args{
				OSString: fmt.Sprintf("!%s", runtime.GOOS),
			},
			want: false,
		},
		{
			name: "NotElligibleExplicitOSDeclaration",
			args: args{
				OSString: "notARealOS",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOSEligible(tt.args.OSString); got != tt.want {
				t.Errorf("isOSEligible() = %v, want %v", got, tt.want)
			}
		})
	}
}
