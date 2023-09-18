package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"
)

type GoFilterDefinition struct {
	FilterDefinition
	Script string `json:"script,omitempty"`
}

type GoFilter struct {
	Filter
	Definition GoFilterDefinition `json:"-"`
}

func GoFilterDefinitionFromObject(id string, obj map[string]interface{}) (*GoFilterDefinition, error) {
	filter := &GoFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	scriptObj, ok := obj["script"]
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "script")
	}
	script, ok := scriptObj.(string)
	if !ok {
		return nil, burrito.WrappedErrorf(
			jsonPropertyTypeError, "script", "string")
	}
	filter.Script = script
	return filter, nil
}

func (f *GoFilter) run(context RunContext) error {
	// Run filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"go",
			append([]string{
				"run",
				context.AbsoluteLocation + string(os.PathSeparator) +
					f.Definition.Script},
				f.Arguments...,
			),
			context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath),
			ShortFilterName(f.Id),
		)
		if err != nil {
			return burrito.WrapError(err, runSubProcessError)
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"go",
			append([]string{
				"run",
				context.AbsoluteLocation + string(os.PathSeparator) +
					f.Definition.Script,
				string(jsonSettings)},
				f.Arguments...,
			),
			context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath),
			ShortFilterName(f.Id),
		)
		if err != nil {
			return burrito.WrapError(err, runSubProcessError)
		}
	}
	return nil
}

func (f *GoFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, burrito.PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *GoFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, burrito.WrapError(err, filterFromObjectError)
	}
	filter := &GoFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *GoFilterDefinition) Check(context RunContext) error {
	_, err := exec.LookPath("go")
	if err != nil {
		return burrito.WrapError(
			err, "Go not found, download and install it from"+
				"https://golang.org/dl/")
	}
	cmd, err := exec.Command("go", "version").Output()
	if err != nil {
		return burrito.WrapError(err, "Failed to check Go version")
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "v")
	Logger.Debugf("Found Go version %s", a)
	return nil
}

func (f *GoFilterDefinition) InstallDependencies(
	parent *RemoteFilterDefinition, dotRegolithPath string,
) error {
	installLocation := ""
	// Install dependencies
	if parent != nil {
		installLocation = parent.GetDownloadPath(dotRegolithPath)
	}
	Logger.Infof("Downloading dependencies for %s...", f.Id)
	joinedPath := filepath.Join(installLocation, f.Script)
	scriptPath, err := filepath.Abs(joinedPath)
	if err != nil {
		return burrito.WrapErrorf(err, filepathAbsError, joinedPath)
	}

	// Install the filter dependencies
	filterPath := filepath.Dir(scriptPath)
	err = RunSubProcess(
		"go",
		[]string{"mod", "download"},
		filterPath,
		"",
		ShortFilterName(f.Id),
	)
	if err != nil {
		return burrito.WrapErrorf(err, "Failed to download Go dependencies", f.Id)
	}
	return nil
}

func (f *GoFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}
