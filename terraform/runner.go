package terraform

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/mevansam/goutils/run"
	"github.com/mevansam/goutils/streams"
)

type Input struct {
	Optional bool
}

type Output struct {
	Sensitive bool
	Type      interface{}
	Value     interface{}
}

type Runner struct {
	cli run.CLI

	// Terraform configuration path
	configPath string

	// Pre-installed plugin path
	pluginPath string

	// Config input variables
	configInputs map[string]Input

	// Additional environment variables to be
	// passed along to terraform CLI commands
	env []string

	// Terraform backend configuration for recipe
	backEnd []string
}

const tfPlanFileName = `tf.plan`

// in: cli - a CLI instance for the running the Terraform binary
// in: configPath - the Terraform configuration path
// in: configInputs - list of input variables expected by the Terraform configuration
// out: an instance of a runner that can be used to execute Terraform
func NewRunner(
	cli run.CLI,
	configPath string,
	pluginPath string,
	configInputs map[string]Input,
) *Runner {

	runner := &Runner{
		cli: cli,

		configPath:   configPath,
		pluginPath:   pluginPath,
		configInputs: configInputs,

		env:     []string{},
		backEnd: []string{},
	}

	return runner
}

func (r *Runner) SetEnv(
	env map[string]string,
) {

	r.env = []string{}
	for k, v := range env {
		r.env = append(r.env, fmt.Sprintf("%s=%s", k, v))
	}
}

func (r *Runner) SetBackend(
	env map[string]string,
) {

	r.backEnd = []string{}
	for k, v := range env {
		r.backEnd = append(r.backEnd, fmt.Sprintf("-backend-config=%s=%s", k, v))
	}
}

func (r *Runner) Init() error {

	argList := []string{"init"}
	if len(r.pluginPath) > 0 {
		argList = append(argList, fmt.Sprintf("-plugin-dir=%s", r.pluginPath))
	}
	argList = append(argList, r.backEnd...)
	argList = append(argList, r.configPath)

	return r.cli.RunWithEnv(argList, r.env)
}

func (r *Runner) Plan(
	args map[string]string,
) error {

	var (
		err     error
		argList []string
	)

	if argList, err = r.prepareArgList(
		args,
		[]string{
			"plan",
			"-input=false",
			fmt.Sprintf(
				"-out=%s",
				filepath.Join(r.cli.WorkingDirectory(), tfPlanFileName),
			),
		},
		r.configPath,
	); err != nil {
		return err
	}

	err = r.cli.RunWithEnv(argList, r.env)
	return err
}

func (r *Runner) Apply(
	args map[string]string,
) (map[string]Output, error) {

	var (
		err    error
		filter streams.Filter
	)

	// create plan if it does not exist
	planPath := filepath.Join(r.cli.WorkingDirectory(), tfPlanFileName)
	if _, err = os.Stat(planPath); os.IsNotExist(err) {
		err = r.Plan(args)
	}
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(planPath)

	// filter out any outputs from terraform
	// run which may container sensitive data
	filter.AddExcludeAfterPattern("Apply complete!")
	r.cli.ApplyFilter(&filter)

	if err = r.cli.RunWithEnv(
		[]string{
			"apply",
			tfPlanFileName,
		},
		r.env,
	); err != nil {
		return nil, err
	}
	return r.GetOutput()
}

func (r *Runner) prepareArgList(
	args map[string]string,
	argList []string,
	configPath string,
) ([]string, error) {

	var (
		exists bool
	)

	required := make(map[string]bool)
	for name, input := range r.configInputs {
		if !input.Optional {
			required[name] = true
		}
	}

	for k, v := range args {

		if _, exists = r.configInputs[k]; !exists {
			return nil, fmt.Errorf(
				"the following argument is not known by the templates: %s", k,
			)
		}

		argList = append(append(argList, "-var"), fmt.Sprintf("%s=%s", k, v))

		// remove requried arg if it exists
		delete(required, k)
	}
	if len(required) > 0 {

		// if some required args remain then they
		// were not passed as an argument and the
		// user needs to be informed via an error
		missing := []string{}
		for k := range required {
			missing = append(missing, k)
		}
		return nil,
			fmt.Errorf(
				"the following required arguments were not provided: %s",
				strings.Join(missing, ","),
			)
	}
	argList = append(argList, configPath)
	return argList, nil
}

func (r *Runner) GetOutput() (map[string]Output, error) {

	var (
		err    error
		filter streams.Filter
	)

	// eat all output sent to default
	// cli output buffer (i.e. stdout)
	filter.SetBlackHole()
	r.cli.ApplyFilter(&filter)

	// create a pipe to read back the
	// actual output from
	output := make(map[string]Output)
	outputBuffer := r.cli.GetPipedOutputBuffer()

	decoder := json.NewDecoder(outputBuffer)
	decodeError := make(chan error)

	go func() {

		// Decode json output stream piped from CLI output
		// and create map of recipe output name-value pairs

		var (
			err error

			t     json.Token
			name  string
			value Output
		)

		for decoder.More() {
			if t, err = decoder.Token(); err != nil {
				break
			}
			if v := reflect.ValueOf(t); v.Kind() == reflect.String {
				name = t.(string)
				if err = decoder.Decode(&value); err != nil {
					err = fmt.Errorf(
						"error decoding value for output '%s': %s",
						name, err.Error(),
					)
					break
				}
				output[name] = value
			}
		}
		if err != nil && err != io.EOF {

			// drain the buffer as otherwise the multi
			// writer will block indefinitely and cli
			// command execution will not return
			b := make([]byte, 1024)
			var e error
			for e == nil {
				_, e = outputBuffer.Read(b)
			}

			decodeError <- err

		} else {
			decodeError <- nil
		}

	}()

	err = r.cli.RunWithEnv([]string{"output", "-json"}, r.env)
	if err != nil {
		return nil, err
	}

	return output, <-decodeError
}

func (r *Runner) Taint(resources []string) error {

	var (
		err error
	)
	// ensure plan file if it exists is removed
	os.RemoveAll(filepath.Join(r.cli.WorkingDirectory(), tfPlanFileName))

	for _, resource := range resources {
		if err = r.cli.RunWithEnv([]string{"taint", resource}, r.env); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) Destroy() error {
	// ensure plan file if it exists is removed
	os.RemoveAll(filepath.Join(r.cli.WorkingDirectory(), tfPlanFileName))

	return r.cli.RunWithEnv([]string{"destroy", "-auto-approve"}, r.env)
}
