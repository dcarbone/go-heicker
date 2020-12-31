package confinator

import (
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

func buildFlagVarTypeKey(varType reflect.Type) string {
	const kfmt = "%s%s%s"
	return fmt.Sprintf(kfmt, varType.PkgPath(), varType.Name(), varType.String())
}

type FlagVarTypeHandlerFunc func(fs *flag.FlagSet, varPtr interface{}, name, usage string)

// DefaultFlagVarTypes returns a list of
func DefaultFlagVarTypes() map[string]FlagVarTypeHandlerFunc {
	kfn := func(ptr interface{}) string {
		return buildFlagVarTypeKey(reflect.TypeOf(ptr))
	}
	return map[string]FlagVarTypeHandlerFunc{
		// *string
		kfn(new(string)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			var v string
			if varPtr.(*string) != nil {
				v = *varPtr.(*string)
			}
			fs.StringVar(varPtr.(*string), name, v, usage)
		},
		// *bool
		kfn(new(bool)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			var v bool
			if varPtr.(*bool) != nil {
				v = *varPtr.(*bool)
			}
			fs.BoolVar(varPtr.(*bool), name, v, usage)
		},
		// *int
		kfn(new(int)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			var v int
			if varPtr.(*int) != nil {
				v = *varPtr.(*int)
			}
			fs.IntVar(varPtr.(*int), name, v, usage)
		},
		// *int64
		kfn(new(int64)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			var v int64
			if varPtr.(*int64) != nil {
				v = *varPtr.(*int64)
			}
			fs.Int64Var(varPtr.(*int64), name, v, usage)
		},
		// *uint
		kfn(new(uint)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			var v uint
			if varPtr.(*uint) != nil {
				v = *varPtr.(*uint)
			}
			fs.UintVar(varPtr.(*uint), name, v, usage)
		},
		// *uint64
		kfn(new(uint64)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			var v uint64
			if varPtr.(*uint64) != nil {
				v = *varPtr.(*uint64)
			}
			fs.Uint64Var(varPtr.(*uint64), name, v, usage)
		},
		// *[]string
		kfn(new([]string)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			fs.Var(newStringSliceValue(varPtr.(*[]string)), name, usage)
		},
		// *[]int
		kfn(new([]int)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			fs.Var(newIntSliceValue(varPtr.(*[]int)), name, usage)
		},
		// *[]uint
		kfn(new([]uint)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			fs.Var(newUintSliceValue(varPtr.(*[]uint)), name, usage)
		},
		// *[]map[string]string
		kfn(new(map[string]string)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			fs.Var(newStringMapValue(varPtr.(*map[string]string)), name, usage)
		},
		// *time.Duration
		kfn(new(time.Duration)): func(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
			var v time.Duration
			if varPtr.(*time.Duration) != nil {
				v = *varPtr.(*time.Duration)
			}
			fs.DurationVar(varPtr.(*time.Duration), name, v, usage)
		},
	}
}

type Confinator struct {
	mu    sync.RWMutex
	types map[string]FlagVarTypeHandlerFunc
}

func NewConfinator() *Confinator {
	cf := new(Confinator)
	cf.types = DefaultFlagVarTypes()
	return cf
}

func (cf *Confinator) RegisterFlagVarType(varPtr interface{}, fn FlagVarTypeHandlerFunc) {
	cf.mu.Lock()
	defer cf.mu.Unlock()
	varType := reflect.TypeOf(varPtr)
	if varType.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("Must provided a pointer, saw %T", varPtr))
	}
	cf.types[buildFlagVarTypeKey(varType)] = fn
}

// stringMapValue is pulled from https://github.com/hashicorp/consul/blob/b5abf61963c7b0bdb674602bfb64051f8e23ddb1/agent/config/flagset.go#L118
type stringMapValue map[string]string

func newStringMapValue(p *map[string]string) *stringMapValue {
	*p = map[string]string{}
	return (*stringMapValue)(p)
}

func (s *stringMapValue) Set(val string) error {
	p := strings.SplitN(val, ":", 2)
	k, v := p[0], ""
	if len(p) == 2 {
		v = p[1]
	}
	(*s)[k] = v
	return nil
}

func (s *stringMapValue) Get() interface{} {
	return s
}

func (s *stringMapValue) String() string {
	var x []string
	for k, v := range *s {
		if v == "" {
			x = append(x, k)
		} else {
			x = append(x, k+":"+v)
		}
	}
	return strings.Join(x, " ")
}

// stringSliceValue is pulled from https://github.com/hashicorp/consul/blob/b5abf61963c7b0bdb674602bfb64051f8e23ddb1/agent/config/flagset.go#L183
type stringSliceValue []string

func newStringSliceValue(p *[]string) *stringSliceValue {
	return (*stringSliceValue)(p)
}

func (s *stringSliceValue) Set(val string) error {
	*s = append(*s, val)
	return nil
}

func (s *stringSliceValue) Get() interface{} {
	return s
}

func (s *stringSliceValue) String() string {
	return strings.Join(*s, " ")
}

type intSliceValue []int

func newIntSliceValue(p *[]int) *intSliceValue {
	return (*intSliceValue)(p)
}

func (s *intSliceValue) Set(val string) error {
	if i, err := strconv.Atoi(val); err != nil {
		return err
	} else {
		*s = append(*s, i)
		return nil
	}
}

func (s *intSliceValue) Get() interface{} {
	return s
}

func (s *intSliceValue) String() string {
	l := len(*s)
	tmp := make([]string, l, l)
	for i, v := range *s {
		tmp[i] = strconv.Itoa(v)
	}
	return strings.Join(tmp, " ")
}

type uintSliceValue []uint

func newUintSliceValue(p *[]uint) *uintSliceValue {
	return (*uintSliceValue)(p)
}

func (s *uintSliceValue) Set(val string) error {
	if i, err := strconv.ParseUint(val, 10, 64); err != nil {
		return err
	} else {
		*s = append(*s, uint(i))
		return nil
	}
}

func (s *uintSliceValue) Get() interface{} {
	return s
}

func (s *uintSliceValue) String() string {
	l := len(*s)
	tmp := make([]string, l, l)
	for i, v := range *s {
		tmp[i] = strconv.FormatUint(uint64(v), 10)
	}
	return strings.Join(tmp, " ")
}

// FlagVar is a convenience method that handles a few common config struct -> flag cases
func (cf *Confinator) FlagVar(fs *flag.FlagSet, varPtr interface{}, name, usage string) {
	cf.mu.RLock()
	defer cf.mu.RUnlock()
	if fn, ok := cf.types[buildFlagVarTypeKey(reflect.TypeOf(varPtr))]; ok {
		fn(fs, varPtr, name, usage)
	} else {
		panic(fmt.Sprintf("invalid type: %T", varPtr))
	}
}

type HelpTextState struct {
	FlagSet        *flag.FlagSet
	LongestName    int
	LongestDefault int
	Current        string
}

type FlagHelpHeaderFunc func(state HelpTextState) string

var DefaultFlagHelpHeaderFunc FlagHelpHeaderFunc = func(state HelpTextState) string {
	return fmt.Sprintf("%s\n", state.FlagSet.Name())
}

type FlagHelpTableHeaderFunc func(state HelpTextState) string

var DefaultFlagHelpTableHeaderFunc FlagHelpTableHeaderFunc = func(state HelpTextState) string {
	return fmt.Sprintf(
		"\t[Flag]%s[Default]%s[Usage]",
		strings.Repeat(" ", state.LongestName-1),
		strings.Repeat(" ", state.LongestDefault-5),
	)
}

type FlagHelpTableRowFunc func(flagNum int, f *flag.Flag, state HelpTextState) string

var DefaultFlagHelpTableRowFunc FlagHelpTableRowFunc = func(flagNum int, f *flag.Flag, state HelpTextState) string {
	return fmt.Sprintf(
		"\n\t-%s%s%s%s%s",
		f.Name,
		strings.Repeat(" ", state.LongestName-len(f.Name)+4),
		f.DefValue,
		strings.Repeat(" ", state.LongestDefault-len(f.DefValue)+4),
		f.Usage,
	)
}

type FlagHelpTableFooterFunc func(state HelpTextState) string

var DefaultFlagHelpTableFooterFunc FlagHelpTableFooterFunc = func(state HelpTextState) string {
	return ""
}

type FlagHelpFooterFunc func(state HelpTextState) string

var DefaultFlagHelpFooterFunc FlagHelpFooterFunc = func(state HelpTextState) string {
	return ""
}

type FlagHelpTextConf struct {
	FlagSet         *flag.FlagSet
	HeaderFunc      FlagHelpHeaderFunc
	TableHeaderFunc FlagHelpTableHeaderFunc
	TableRowFunc    FlagHelpTableRowFunc
	TableFooterFunc FlagHelpTableFooterFunc
	FooterFunc      FlagHelpFooterFunc
}

func FlagHelpText(conf FlagHelpTextConf) string {
	var (
		longestName    int
		longestDefault int
		out            string

		hf  FlagHelpHeaderFunc
		thf FlagHelpTableHeaderFunc
		trf FlagHelpTableRowFunc
		tff FlagHelpTableFooterFunc
		ff  FlagHelpFooterFunc

		fs = conf.FlagSet
	)

	if conf.HeaderFunc == nil {
		hf = DefaultFlagHelpHeaderFunc
	} else {
		hf = conf.HeaderFunc
	}
	if conf.TableHeaderFunc == nil {
		thf = DefaultFlagHelpTableHeaderFunc
	} else {
		thf = conf.TableHeaderFunc
	}
	if conf.TableRowFunc == nil {
		trf = DefaultFlagHelpTableRowFunc
	} else {
		trf = conf.TableRowFunc
	}
	if conf.TableFooterFunc == nil {
		tff = DefaultFlagHelpTableFooterFunc
	} else {
		tff = conf.TableFooterFunc
	}
	if conf.FooterFunc == nil {
		ff = DefaultFlagHelpFooterFunc
	} else {
		ff = conf.FooterFunc
	}

	fs.VisitAll(func(f *flag.Flag) {
		if l := len(f.Name); l > longestName {
			longestName = l
		}
		if l := len(f.DefValue); l > longestDefault {
			longestDefault = l
		}
	})

	makeState := func() HelpTextState {
		return HelpTextState{
			FlagSet:        fs,
			LongestName:    longestName,
			LongestDefault: longestDefault,
			Current:        out,
		}
	}

	out = hf(makeState())
	out = fmt.Sprintf("%s%s", out, thf(makeState()))
	i := 0
	fs.VisitAll(func(f *flag.Flag) {
		out = fmt.Sprintf("%s%s", out, trf(i, f, makeState()))
		i++
	})
	out = fmt.Sprintf("%s%s%s", out, tff(makeState()), ff(makeState()))

	return out
}
